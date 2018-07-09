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

package main

import (
	"github.com/daocloud/anchor/anchor-ipam/backend/allocator"
	"github.com/daocloud/anchor/anchor-ipam/backend/etcd"
	"github.com/daocloud/anchor/anchor-ipam/k8s"
	"github.com/coreos/etcd/pkg/transport"
	"net"
	"strings"
	"fmt"

	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/types"
	"github.com/containernetworking/cni/pkg/types/current"
	"github.com/containernetworking/cni/pkg/version"
)

// TODO: logging and debug.
func main() {
	skel.PluginMain(cmdAdd, cmdDel, version.All)
}

// TODO: create a seperate function named etcdClient which return kv and mutex.
func cmdAdd(args *skel.CmdArgs) error {
	ipamConf, confVersion, err := allocator.LoadIPAMConfig(args.StdinData, args.Args)
	if err != nil {
		return err
	}
	result := &current.Result{}

	tlsInfo := &transport.TLSInfo{
		CertFile:      ipamConf.CertFile,
		KeyFile:       ipamConf.KeyFile,
		TrustedCAFile: ipamConf.TrustedCAFile,
	}
	tlsConfig, _ := tlsInfo.ClientConfig()

	store, err := etcd.New(ipamConf.Name, strings.Split(ipamConf.Endpoints, ","), tlsConfig)
	defer store.Close()
	if err != nil {
		return err
	}

	// Get annotations of the pod, such as ipAddrs and current user.

	// 1. Get conf for k8s client and create a k8s_client
	k8sClient, err := k8s.NewK8sClient(ipamConf.Kubernetes, ipamConf.Policy)
	if err != nil {
		return err
	}

	// 2. Get K8S_POD_NAME and K8S_POD_NAMESPACE.
	k8sArgs := k8s.K8sArgs{}
	if err := types.LoadArgs(args.Args, &k8sArgs); err != nil {
		return err
	}

	// 3. Get annotations from k8s_client via K8S_POD_NAME and K8S_POD_NAMESPACE.
	label, annot, err := k8s.GetK8sPodInfo(k8sClient, string(k8sArgs.K8S_POD_NAME), string(k8sArgs.K8S_POD_NAMESPACE))
	if err != nil {
		return fmt.Errorf("Error while read annotaions for pod" + err.Error())
	}

	userDefinedSubnet := annot["cni.daocloud.io/subnet"]
	userDefinedRoutes := annot["cni.daocloud.io/routes"]
	userDefinedGateway := annot["cni.daocloud.io/gateway"]

	app := label["io.daocloud.dce.app"]
	service := label["io.daocloud.dce.name"]

	if app == "" {
		app = "unknown"
	}
	if service == "" {
		service = "unknown"
	}

	if userDefinedSubnet == "" {
		return fmt.Errorf("No ip found for pod " + string(k8sArgs.K8S_POD_NAME))
	}

	_, subnet, err := net.ParseCIDR(userDefinedSubnet)
	if err != nil {
		return err
	}

	if userDefinedGateway != "" {
		gw := types.Route{
			Dst: net.IPNet{
				IP:   net.IPv4zero,
				Mask: net.IPv4Mask(0, 0, 0, 0),
			},
			GW: net.ParseIP(userDefinedGateway),
		}
		result.Routes = append(result.Routes, &gw)
	}

	if userDefinedRoutes != "" {
		routes := strings.Split(userDefinedRoutes, ";")
		for _, r := range routes {
			_, dst, _ := net.ParseCIDR(strings.Split(r, ",")[0])
			gateway := strings.Split(r, ",")[1]

			gw := types.Route{
				Dst: *dst,
				GW: net.ParseIP(gateway),
			}
			result.Routes = append(result.Routes, &gw)
		}
	}
	// result.Routes = append(result.Routes, ipamConf.Routes...)

	if ipamConf.Service_IPNet != "" {
		_, service_net, err := net.ParseCIDR(ipamConf.Service_IPNet)
		if err != nil {
			return fmt.Errorf("Invalid service cluster ip range: " + ipamConf.Service_IPNet)
		}
		for _, node_ip := range ipamConf.Node_IPs {
			if subnet.Contains(net.ParseIP(node_ip)) {
				sn := types.Route{
					Dst: *service_net,
					GW: net.ParseIP(node_ip),
				}
				result.Routes = append(result.Routes, &sn)
				break
			}
			// If none of node_ip in subnet, nothing to do.
		}
	}

	userDefinedNameserver  :=  annot["cni.daocloud.io/nameserver"]
	userDefinedDomain      :=  annot["cni.daocloud.io/domain"]
	userDefinedSearch      :=  annot["cni.daocloud.io/search"]
	userDefinedOptions     :=  annot["cni.daocloud.io/options"]
	dns, err := generateDNS(userDefinedNameserver, userDefinedDomain, userDefinedSearch, userDefinedOptions)
	result.DNS = *dns

	alloc := allocator.NewAnchorAllocator(subnet, store, string(k8sArgs.K8S_POD_NAME), string(k8sArgs.K8S_POD_NAMESPACE), app, service)

	ipConf, err := alloc.Get(args.ContainerID)
	if err != nil {
		return err
	}

	// Below here, if error, we should call store.Release(args.ContainerID) to release the IP written to database.
	if userDefinedGateway == "" {
		gw := types.Route{
			Dst: net.IPNet{
				IP:   net.IPv4zero,
				Mask: net.IPv4Mask(0, 0, 0, 0),
			},
			GW: ipConf.Gateway,
		}
		result.Routes = append(result.Routes, &gw)
	}
	result.IPs = append(result.IPs, ipConf)

	return types.PrintResult(result, confVersion)
}

func cmdDel(args *skel.CmdArgs) error {
	ipamConf, _, err := allocator.LoadIPAMConfig(args.StdinData, args.Args)
	if err != nil {
		return err
	}

	tlsInfo := &transport.TLSInfo{
		CertFile:      ipamConf.CertFile,
		KeyFile:       ipamConf.KeyFile,
		TrustedCAFile: ipamConf.TrustedCAFile,
	}
	tlsConfig, _ := tlsInfo.ClientConfig()

	store, err := etcd.New(ipamConf.Name, strings.Split(ipamConf.Endpoints, ","), tlsConfig)
	defer store.Close()
	if err != nil {
		return err
	}
	store.Lock()
	defer store.Unlock()
	return store.Release(args.ContainerID)
	// TODO: allocator and deleter.
}
