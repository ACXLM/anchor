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
	"encoding/json"
	"fmt"
	"github.com/containernetworking/cni/pkg/types"
	"github.com/daocloud/anchor/anchor-ipam/k8s"
	"net"
)

// The top-level network config, just so we can get the IPAM block
type Net struct {
	Name       string      `json:"name"`
	CNIVersion string      `json:"cniVersion"`
	Type       string      `json:"type"`
	Master     string      `json:"master"`
	IPAM       *IPAMConfig `json:"ipam"`
	// TODO:
	// LogLevel       string         `json:"log_level"`
}

// IPAMConfig represents the IP related network configuration.
// This nests Range because we initially only supported a single
// range directly, and wish to preserve backwards compatability
type IPAMConfig struct {
	Name string
	Type string                  `json:"type"`
	// etcd client
	Endpoints     string         `json:"etcd_endpoints"`
	// Used for k8s client
	Kubernetes    k8s.Kubernetes `json:"kubernetes"`
	Policy        k8s.Policy     `json:"policy"`
	// etcd perm files
	CertFile      string         `json:"etcd_cert_file"`
	KeyFile       string         `json:"etcd_key_file"`
	TrustedCAFile string         `json:"etcd_ca_cert_file"`
	Service_IPNet string         `json:"service_ipnet"`
	Node_IPs      []string       `json:"node_ips"`
	// additional network config for pods
	Routes        []*types.Route `json:"routes,omitempty"`
	ResolvConf    string         `json:"resolvConf,omitempty"`

	// Args       *struct {
	//       A *IPAMArgs `json:"cni"`
	// } `json:"args"`
	// IPv4Pools  []string `json:"ipv4_pools,omitempty"`
	// IPv6Pools  []string `json:"ipv6_pools,omitempty"`
	// Subnet     string   `json:"subnet"`
	// EtcdAuthority  string     `json:"etcd_authority"`
}

type IPAMEnvArgs struct {
	types.CommonArgs
	IP net.IP `json:"ip,omitempty"`
}

type IPAMArgs struct {
	IPs []net.IP `json:"ips"`
}

type RangeSet []Range

type Range struct {
	RangeStart net.IP      `json:"rangeStart,omitempty"` // The first ip, inclusive
	RangeEnd   net.IP      `json:"rangeEnd,omitempty"`   // The last ip, inclusive
	Subnet     types.IPNet `json:"subnet"`
	Gateway    net.IP      `json:"gateway,omitempty"`
}

// NewIPAMConfig creates a NetworkConfig from the given network name.
func LoadIPAMConfig(bytes []byte, envArgs string) (*IPAMConfig, string, error) {
	n := Net{}
	if err := json.Unmarshal(bytes, &n); err != nil {
		return nil, "", err
	}

	if n.IPAM == nil {
		return nil, "", fmt.Errorf("IPAM config missing 'ipam' key")
	}

	if n.IPAM.Endpoints == "" {
		return nil, "", fmt.Errorf("IPAM config missing 'etcd_endpoints' keys")
	}

	/*
		if n.IPAM.Kubernetes == nil {
			return nil, "", fmt.Errorf("IPAM config missing 'kubernetes' keys")
		}

		if n.IPAM.Policy == nil {
			return nil, "", fmt.Errorf("IPAM config missing 'policy' keys")
		}


		// Parse custom IP from both env args *and* the top-level args config
		if envArgs != "" {
			e := IPAMEnvArgs{}
			err := types.LoadArgs(envArgs, &e)
			if err != nil {
				return nil, "", err
			}

			if e.IP != nil {
				n.IPAM.IPArgs = []net.IP{e.IP}
			}
		}

		if n.Args != nil && n.Args.A != nil && len(n.Args.A.IPs) != 0 {
			n.IPAM.IPArgs = append(n.IPAM.IPArgs, n.Args.A.IPs...)
		}

		for idx, _ := range n.IPAM.IPArgs {
			if err := canonicalizeIP(&n.IPAM.IPArgs[idx]); err != nil {
				return nil, "", fmt.Errorf("cannot understand ip: %v", err)
			}
		}

		// Copy net name into IPAM so not to drag Net struct around
		n.IPAM.Name = n.Name
	*/
	return n.IPAM, n.CNIVersion, nil
}
