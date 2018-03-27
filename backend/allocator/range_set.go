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
)

// Contains returns true if any range in this set contains an IP
func (s *RangeSet) Contains(addr net.IP) bool {
	r, _ := s.RangeFor(addr)
	return r != nil
}

// IsSubset returns true if s is a subset of s1.
func (s *RangeSet) IsSubset(s1 *RangeSet) bool {
	l := len(*s1)
	for _, r := range *s {
		cnt := 0
		for _, r1 := range *s1 {
			if r.IsSubset(&r1) {
				break
			}
			cnt++
		}
		if cnt == l-1 {
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

// LoadRangeSet loads RangeSet from string whose format likes "10.0.0.[2-4]; 10.0.1.4"
// We don't init subnet and gateway here because only few ips are returned for every time,
// and only those need subnet and gateway information.
func LoadRangeSet(ipAddrs string) *RangeSet {
	ret := RangeSet{}
	ranges := strings.Split(ipAddrs, ";")
	for _, r := range ranges {
		// TODO:
		if strings.HasSuffix(r, "]") {
			// Example of segments: ["10.0.1.", "4-8"]
			segments := strings.Split(strings.TrimSuffix(r, "]"), "[")
			suffixs := strings.Split(segments[1], "-")
			ret = append(ret, Range{
				RangeStart: net.ParseIP(segments[0] + suffixs[0]),
				RangeEnd:   net.ParseIP(segments[0] + suffixs[1]),
			})
		} else {
			ret = append(ret, Range{
				RangeStart: net.ParseIP(r),
				RangeEnd:   net.ParseIP(r),
			})
		}
	}
	return &ret
}
