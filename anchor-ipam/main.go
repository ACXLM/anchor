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

	// TODO: test
	if ipamConf.ResolvConf != "" {
		dns, err := parseResolvConf(ipamConf.ResolvConf)
		if err != nil {
			return err
		}
		result.DNS = *dns
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




	userDefinedSubnet := annot["cni.daocloud.io/subnet"] // "10.0.0.[11-14],10.0.1.2"
	userDefinedRoutes := annot["cni.daocloud.io/routes"]
	userDefinedGateway := annot["cni.daocloud.io/gateway"]

	userDefinedNameserver := annot["cni.daocloud.io/nameserver"]
	userDefinedDomain := annot["cni.daocloud.io/domain"]
	userDefinedSearch := annot["cni.daocloud.io/search"]
	userDefinedOptions := annot["cni.daocloud.io/option"]

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
	alloc := allocator.NewAnchorAllocator(subnet, store, string(k8sArgs.K8S_POD_NAME), string(k8sArgs.K8S_POD_NAMESPACE), app, service)

	ipConf, err := alloc.Get(args.ContainerID)
	if err != nil {
		// TODO:
		return err
	}

	result.IPs = append(result.IPs, ipConf)

	if userDefinedGateway != "" {
		// TODO: check invalid format
		gw := types.Route{
			Dst: net.IPNet{
				IP:   net.IPv4zero,
				Mask: net.IPv4Mask(0, 0, 0, 0),
			},
			GW: net.ParseIP(userDefinedGateway),
		}
		result.Routes = append(result.Routes, &gw)
	} else {
		gw := types.Route{
			Dst: net.IPNet{
				IP:   net.IPv4zero,
				Mask: net.IPv4Mask(0, 0, 0, 0),
			},
			GW: ipConf.Gateway,
		}
		result.Routes = append(result.Routes, &gw)
	}

	if userDefinedRoutes != "" {
		// TODO: check invalid format
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

	// dns := types.DNS{}
	if userDefinedNameserver != "" {
		servers :=  strings.Split(userDefinedNameserver, ",")
		result.DNS.Nameservers = append(result.DNS.Nameservers, servers...)
	}

	if userDefinedDomain != "" {
		result.DNS.Domain = userDefinedDomain
	}

	if userDefinedSearch != "" {
		result.DNS.Search = append(result.DNS.Search, strings.Fields(userDefinedSearch)...)
	}
	if userDefinedOptions != "" {
		result.DNS.Options = append(result.DNS.Options, strings.Fields(userDefinedOptions)...)
	}

	result.Routes = append(result.Routes, ipamConf.Routes...)
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
