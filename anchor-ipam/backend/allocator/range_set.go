// Copyright 2017 CNI authors
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
	"strings"
	"sort"

	"github.com/containernetworking/plugins/pkg/ip"
)

// Contains returns true if any range in this set contains an IP
func (s *RangeSet) Contains(addr net.IP) bool {
	r, _ := s.RangeFor(addr)
	return r != nil
}

// IsSubset returns true if s is a subset of s1.
// TODO: bug when s1 is nil or nil-nil.
func (s *RangeSet) IsSubset(s1 *RangeSet) bool {
	l := len(*s1)
	for _, r := range *s {
		cursor := 0
		for _, r1 := range *s1 {
			if r.IsSubset(&r1) {
				break
			}
			cursor++
		}
		if cursor == l {
			return false
		}
	}

	return true
}

// RangeFor finds the range that contains an IP, or nil if not found
func (s *RangeSet) RangeFor(addr net.IP) (*Range, error) {
	if err := canonicalizeIP(&addr); err != nil {
		return nil, err
	}

	for _, r := range *s {
		if r.Contains(addr) {
			return &r, nil
		}
	}

	return nil, fmt.Errorf("%s not in range set %s", addr.String(), s.String())
}

// Overlaps returns true if any ranges in any set overlap with this one
func (s *RangeSet) Overlaps(p1 *RangeSet) bool {
	for _, r := range *s {
		for _, r1 := range *p1 {
			if r.Overlaps(&r1) {
				return true
			}
		}
	}
	return false
}

// Canonicalize ensures the RangeSet is in a standard form, and detects any
// invalid input. Call Range.Canonicalize() on every Range in the set
func (s *RangeSet) Canonicalize() error {
	if len(*s) == 0 {
		return fmt.Errorf("empty range set")
	}

	fam := 0
	for i, _ := range *s {
		if err := (*s)[i].Canonicalize(); err != nil {
			return err
		}
		if i == 0 {
			fam = len((*s)[i].RangeStart)
		} else {
			if fam != len((*s)[i].RangeStart) {
				return fmt.Errorf("mixed address families")
			}
		}
	}

	// Make sure none of the ranges in the set overlap
	l := len(*s)
	for i, r1 := range (*s)[:l-1] {
		for _, r2 := range (*s)[i+1:] {
			if r1.Overlaps(&r2) {
				return fmt.Errorf("subnets %s and %s overlap", r1.String(), r2.String())
			}
		}
	}

	return nil
}

func (s *RangeSet) String() string {
	out := []string{}
	for _, r := range *s {
		out = append(out, r.String())
	}

	return strings.Join(out, ",")
}


func (s RangeSet) Len() int {
	return len(s)
}

func (s RangeSet) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s RangeSet) Less(i, j int) bool {
	a, b := s[i], s[j]

	if ip.Cmp(a.RangeStart, b.RangeStart) != 0 {
		return ip.Cmp(a.RangeStart, b.RangeStart) < 0
	}
	return ip.Cmp(a.RangeEnd, b.RangeEnd) < 0
}

// LoadRangeSet loads RangeSet from string. eg: "10.0.0.[2-4], 10.0.1.4, 10.0.1.5, 10.0.1.9",
// this func return RangeSet with 3 ranges contained. No subnet and gateway information here.
func LoadRangeSet(ipAddrs string) (*RangeSet, error) {
	if ipAddrs == "" {
		return nil, fmt.Errorf("Input of IP ranges is empty")
	}
	ret := RangeSet{}
	ranges := strings.Split(ipAddrs, ",")

	for _, r := range ranges {
		// Remove all lead blanks and tailed blanks.
		r = strings.TrimSpace(r)
		if strings.HasSuffix(r, "]") {
			// eg: ["10.0.1.", "4-8"]
			segments := strings.Split(strings.TrimSuffix(r, "]"), "[")
			suffixs := strings.Split(segments[1], "-")

			start := net.ParseIP(segments[0] + suffixs[0])
			end := net.ParseIP(segments[0] + suffixs[1])
			if start == nil || end == nil {
				return nil, fmt.Errorf("Input of IP ranges is valid")
			}

			ret = append(ret, Range{
				RangeStart: start,
				RangeEnd: end,
			})
		} else {
			// eg: 10.1.8.9
			current := net.ParseIP(r)
			if current == nil {
				return nil, fmt.Errorf("Input of IP range is valid")
			}
			ret = append(ret, Range{
				RangeStart: current,
				RangeEnd: current,
			})
		}
	}
	sort.Sort(ret)

	cursor := 0
	l := len(ret)
	i := 1
	for ; i < l; i++ {
		// A gap here
		if ip.Cmp(ret[cursor].RangeEnd, ret[i].RangeStart) < 0 && !ret[i].RangeStart.Equal(ip.NextIP(ret[cursor].RangeEnd)) {
			if i > cursor + 1 {
				ret = append(ret[:cursor + 1], ret[i:]...)
				gap := i - cursor - 1
				l -= gap
				i -= gap

			}
			cursor += 1
		} else { // overlap case
			if ip.Cmp(ret[i].RangeEnd, ret[cursor].RangeEnd) > 0 {
				ret[cursor].RangeEnd = ret[i].RangeEnd
			}
		}
	}

	if i > cursor + 1 {
		ret = append(ret[:cursor + 1], ret[i:]...)
	}
	return &ret, nil
}
