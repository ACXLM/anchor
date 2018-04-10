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
	"net"
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
	a.store.Lock()
	defer a.store.Unlock()
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
		// TODO: Use iter
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
