// Copyright 2015 CNI authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package allocator

import (
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/containernetworking/cni/pkg/types/current"
	"github.com/containernetworking/plugins/pkg/ip"
	"github.com/daocloud/anchor/anchor-ipam/backend"
)

type IPAllocator struct {
	rangeset *RangeSet
	store    backend.Store
	rangeID  string // Used for tracking last reserved ip
}

type AnchorAllocator struct {
	user         string
	requestedIPs *RangeSet
	store        backend.Store
	podName      string
	podNamespace string
}

func NewAnchorAllocator(user string, requestedIPs *RangeSet, store backend.Store, podName string, podNamespace string) *AnchorAllocator {
	return &AnchorAllocator{
		user:         user,
		requestedIPs: requestedIPs,
		store:        store,
		podName:      podName,
		podNamespace: podNamespace,
	}
}

func NewIPAllocator(s *RangeSet, store backend.Store, id int) *IPAllocator {
	return &IPAllocator{
		rangeset: s,
		store:    store,
		rangeID:  strconv.Itoa(id),
	}
}

func (a *AnchorAllocator) Get(id string) (*current.IPConfig, error) {
	// TODO: minimal lock range.
	// TODO: lock Error while adding to cni network: context canceled
	// a.store.Lock()
	// defer a.store.Unlock()
	var errors []string
	ips, err := a.store.GetOwnedIPs(a.user)
	if err != nil {
		errors = append(errors, "Cannot read owned IP from etcd")
		errors = append(errors, err.Error())
	}

	if errors != nil {
		return nil, fmt.Errorf(strings.Join(errors, ";"))
	}

	ownedIPs := LoadRangeSet(ips)

	if !a.requestedIPs.IsSubset(ownedIPs) {
		return nil, fmt.Errorf("Requested IPs out of range of the available for user %s", a.user)
	}

	usedByPod, err := a.store.GetUsedByPod(a.podName, a.podNamespace)
	if err != nil {
		return nil, err
	}

	// Pick one unused from requestedIPs
	// TODO: RangeFor
	for _, r := range *a.requestedIPs {
		// TODO: Not beautiful.
		// TODO: iter blow
		var iter net.IP
		for iter = r.RangeStart; !iter.Equal(ip.NextIP(r.RangeEnd)); iter = ip.NextIP(iter) {
			avail := true
			for _, used := range usedByPod {
				// if ip.Cmp(iter, used) == 0 {
				if iter.Equal(used) {
					avail = false
					break
				}
			}
			if avail {
				// Get subnet and gateway information
				subnet, gw, err := a.store.GetGatewayForIP(iter)
				if err != nil {
					errors = append(errors, err.Error())
					continue
				}

				if iter.Equal(*gw) {
					errors = append(errors, "gateway")
					continue

				}
				_, err = a.store.Reserve(id, iter, a.podName, a.podNamespace)
				if err != nil {
					// TODO: log
					errors = append(errors, "Cannot write to db")
					continue
				}

				return &current.IPConfig{
					Version: "4",
					Address: net.IPNet{IP: iter, Mask: subnet.Mask},
					Gateway: *gw,
				}, nil
			}
		}
	}
	errors = append(errors, "No available IP")
	errors = append(errors, ownedIPs.String())
	errors = append(errors, a.requestedIPs.String())

	return nil, fmt.Errorf(strings.Join(errors, ";"))

	// return nil, fmt.Errorf("No available IP for requested IPs")
}

// Get alocates an IP
/*
func (a *IPAllocator) Get(id string, requestedIP net.IP) (*current.IPConfig, error) {
	a.store.Lock()
	defer a.store.Unlock()

	var reservedIP *net.IPNet
	var gw net.IP

	if requestedIP != nil {
		if err := canonicalizeIP(&requestedIP); err != nil {
			return nil, err
		}

		r, err := a.rangeset.RangeFor(requestedIP)
		if err != nil {
			return nil, err
		}

		if requestedIP.Equal(r.Gateway) {
			return nil, fmt.Errorf("requested ip %s is subnet's gateway", requestedIP.String())
		}

		reserved, err := a.store.Reserve(id, requestedIP, a.rangeID)
		if err != nil {
			return nil, err
		}

		if !reserved {
			return nil, fmt.Errorf("requested IP address %s is not available in range set %s", requestedIP, a.rangeset.String())
		}
		reservedIP = &net.IPNet{IP: requestedIP, Mask: r.Subnet.Mask}
		gw = r.Gateway

	} else {
		iter, err := a.GetIter()
		if err != nil {
			return nil, err
		}
		for {
			reservedIP, gw = iter.Next()
			if reservedIP == nil {
				break
			}

			reserved, err := a.store.Reserve(id, reservedIP.IP, a.rangeID)
			if err != nil {
				return nil, err
			}
			if reserved {
				break
			}
		}
	}

	if reservedIP == nil {
		return nil, fmt.Errorf("no IP addresses available in range set: %s", a.rangeset.String())
	}
	version := "4"
	if reservedIP.IP.To4() == nil {
		version = "6"
	}

	return &current.IPConfig{
		Version: version,
		Address: *reservedIP,
		Gateway: gw,
	}, nil
}
*/

// Release clears all IPs allocated for the container with given ID
func (a *IPAllocator) Release(id string) error {
	a.store.Lock()
	defer a.store.Unlock()

	return a.store.Release(id)
}

type RangeIter struct {
	rangeset *RangeSet

	// The current range id
	rangeIdx int

	// Our current position
	cur net.IP

	// The IP and range index where we started iterating; if we hit this again, we're done.
	startIP    net.IP
	startRange int
}

// GetIter encapsulates the strategy for this allocator.
// We use a round-robin strategy, attempting to evenly use the whole set.
// More specifically, a crash-looping container will not see the same IP until
// the entire range has been run through.
// We may wish to consider avoiding recently-released IPs in the future.
func (a *IPAllocator) GetIter() (*RangeIter, error) {
	iter := RangeIter{
		rangeset: a.rangeset,
	}

	// Round-robin by trying to allocate from the last reserved IP + 1
	startFromLastReservedIP := false

	// We might get a last reserved IP that is wrong if the range indexes changed.
	// This is not critical, we just lose round-robin this one time.
	lastReservedIP, err := a.store.LastReservedIP(a.rangeID)
	if err != nil && !os.IsNotExist(err) {
		log.Printf("Error retrieving last reserved ip: %v", err)
	} else if lastReservedIP != nil {
		startFromLastReservedIP = a.rangeset.Contains(lastReservedIP)
	}

	// Find the range in the set with this IP
	if startFromLastReservedIP {
		for i, r := range *a.rangeset {
			if r.Contains(lastReservedIP) {
				iter.rangeIdx = i
				iter.startRange = i

				// We advance the cursor on every Next(), so the first call
				// to next() will return lastReservedIP + 1
				iter.cur = lastReservedIP
				break
			}
		}
	} else {
		iter.rangeIdx = 0
		iter.startRange = 0
		iter.startIP = (*a.rangeset)[0].RangeStart
	}
	return &iter, nil
}

// Next returns the next IP, its mask, and its gateway. Returns nil
// if the iterator has been exhausted
func (i *RangeIter) Next() (*net.IPNet, net.IP) {
	r := (*i.rangeset)[i.rangeIdx]

	// If this is the first time iterating and we're not starting in the middle
	// of the range, then start at rangeStart, which is inclusive
	if i.cur == nil {
		i.cur = r.RangeStart
		i.startIP = i.cur
		if i.cur.Equal(r.Gateway) {
			return i.Next()
		}
		return &net.IPNet{IP: i.cur, Mask: r.Subnet.Mask}, r.Gateway
	}

	// If we've reached the end of this range, we need to advance the range
	// RangeEnd is inclusive as well
	if i.cur.Equal(r.RangeEnd) {
		i.rangeIdx += 1
		i.rangeIdx %= len(*i.rangeset)
		r = (*i.rangeset)[i.rangeIdx]

		i.cur = r.RangeStart
	} else {
		i.cur = ip.NextIP(i.cur)
	}

	if i.startIP == nil {
		i.startIP = i.cur
	} else if i.rangeIdx == i.startRange && i.cur.Equal(i.startIP) {
		// IF we've looped back to where we started, give up
		return nil, nil
	}

	if i.cur.Equal(r.Gateway) {
		return i.Next()
	}

	return &net.IPNet{IP: i.cur, Mask: r.Subnet.Mask}, r.Gateway
}
