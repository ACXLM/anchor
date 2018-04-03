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
	// "crypto/tls"
	// "encoding/json"
	// "fmt"
	"github.com/daocloud/anchor/anchor-ipam/backend/allocator"
	"github.com/daocloud/anchor/anchor-ipam/backend/etcd"
	"github.com/daocloud/anchor/anchor-ipam/k8s"
	"github.com/coreos/etcd/pkg/transport"
	"net"
	// "strings"

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

	// Get conf for etcd store.
	/*
		var tlsConfig *tls.Config
		if ipamConf.CertFile != "" &&
			ipamConf.KeyFile != "" &&
			ipamConf.TrustedCAFile != "" {
			tlsInfo := transport.TLSInfo{
				CertFile:      ipamConf.CertFile,
				KeyFile:       ipamConf.KeyFile,
				TrustedCAFile: ipamConf.TrustedCAFile,
			}

			tlsConfig, err = tlsInfo.ClientConfig()
			if err != nil {
				return err
			}
		}
	*/
	store, err := etcd.New(ipamConf.Name, ipamConf.Endpoints, tlsConfig)
	defer store.Close()
	if err != nil {
		return err
	}

	// Get annotations of the pod, such as ipAddrs and current user.

	// 1. Get conf for k8s client and create a k8s_client
	// Conf mainly policy and kuberconfig, those inforamtion is written at /etc/cni/net.d
	/*
		conf := k8s.NetConf{}
		if err := json.Unmarshal(args.StdinData, &conf); err != nil {
			return fmt.Errorf("failed to load netconf: %v", err)
		}
	*/
	// k8sClient, err := k8s.NewK8sClient(conf)
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
	_, annot, err := k8s.GetK8sPodInfo(k8sClient, string(k8sArgs.K8S_POD_NAME), string(k8sArgs.K8S_POD_NAMESPACE))

	ipAddrs := annot["cni.daocloud.io/ipAddrs"] // "10.0.0.[11-14],10.0.1.2"

	currentUser := annot["cni.daocloud.io/currentUser"] // User01

	// TODO:
	if ipAddrs == "" || currentUser == "" {
		return err
	}
	// podName := string(k8sArgs.K8S_POD_NAME)
	// podNamespace := string(k8sArgs.K8S_POD_NAMESPACE)
	alloc := allocator.NewAnchorAllocator(currentUser, allocator.LoadRangeSet(ipAddrs), store, string(k8sArgs.K8S_POD_NAME), string(k8sArgs.K8S_POD_NAMESPACE))

	ipConf, err := alloc.Get(args.ContainerID)
	if err != nil {
		// TODO:
		return err
	}

	result.IPs = append(result.IPs, ipConf)
	gw := types.Route{
		Dst: net.IPNet{
			IP:   net.IPv4zero,
			Mask: net.IPv4Mask(0, 0, 0, 0),
		},
		GW: ipConf.Gateway,
	}
	result.Routes = append(result.Routes, &gw)

	return types.PrintResult(result, confVersion)
}

func cmdDel(args *skel.CmdArgs) error {
	ipamConf, _, err := allocator.LoadIPAMConfig(args.StdinData, args.Args)
	if err != nil {
		return err
	}
	/*
		var tlsConfig *tls.Config
		if ipamConf.CertFile != "" &&
			ipamConf.KeyFile != "" &&
			ipamConf.TrustedCAFile != "" {
			tlsInfo := transport.TLSInfo{
				CertFile: ipamConf.CertFile,
				KeyFile: ipamConf.KeyFile,
				TrustedCAFile: ipamConf.TrustedCAFile,
			}

			tlsConfig, err = tlsInfo.ClientConfig()
			if err != nil {
				return err
			}
		}
	*/
	tlsInfo := &transport.TLSInfo{
		CertFile:      ipamConf.CertFile,
		KeyFile:       ipamConf.KeyFile,
		TrustedCAFile: ipamConf.TrustedCAFile,
	}
	tlsConfig, _ := tlsInfo.ClientConfig()

	store, err := etcd.New(ipamConf.Name, ipamConf.Endpoints, tlsConfig)
	defer store.Close()
	// store, err := etcd.New(ipamConf.Name, ipamConf.Endpoints, tlsConfig)
	if err != nil {
		return err
	}
	return store.Release(args.ContainerID)
	// TODO: allocator and deleter.
	/*
		alloc := allocator.NewAnchorAllocator(&rangeset, store, idx)


		// Loop through all ranges, releasing all IPs, even if an error occurs
		var errors []string
		for idx, rangeset := range ipamConf.Ranges {
			alloc := allocator.NewAnchorAllocator(&rangeset, store, idx)

			err := ipAllocator.Release(args.ContainerID)
			if err != nil {
				errors = append(errors, err.Error())
			}
		}

		if errors != nil {
			return fmt.Errorf(strings.Join(errors, ";"))
		}
		return nil
	*/
}
