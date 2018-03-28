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

package etcd

import (
	"context"
	"fmt"
	// "crypto/tls"
	"github.com/containernetworking/cni/pkg/types"
	"github.com/daocloud/anchor/backend"
	"net"
	"strings"
	"time"
	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/clientv3/concurrency"
)

const lastIPFilePrefix = "last_reserved_ip."

// var defaultDataDir = "/var/lib/cni/networks"
var defaultDataDir = "/ipam"

// Store is a simple etcd-backed store that creates one kv pair per IP
// address. The value of the pair is the container ID.
type Store struct {
	mutex *concurrency.Mutex
	kv    clientv3.KV
}

// Store implements the Store interface
var _ backend.Store = &Store{}

// func New(network string, endPoints []string, tlsConfig *tls.Config) (*Store, error) {
func New(network string, endPoints []string) (*Store, error) {
	if len(endPoints) == 0 {
		return nil, fmt.Errorf("No available endpoints for etcd client")
	}

	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   endPoints,
		DialTimeout: 5 * time.Second,
		// TLS:         tlsConfig,
	})

	if err != nil {
		return nil, err
	}
	// TODO: No, this will bear a bug.
	// defer cli.Close()
	session, err := concurrency.NewSession(cli)
	if err != nil {
		return nil, err
	}

	mutex := concurrency.NewMutex(session, "/ipam/lock")
	kv := clientv3.NewKV(cli)
	return &Store{mutex, kv}, nil
}

func (s *Store) Lock() error {
	return s.mutex.Lock(context.TODO())
}

func (s *Store) Unlock() error {
	return s.mutex.Unlock(context.TODO())
}

func (s *Store) Close() error {
	return nil
	// return s.Unlock()
}

func (s *Store) GetOwnedIPs(user string) (string, error) {
	resp, err := s.kv.Get(context.TODO(), "/ipam/users/"+user)
	if err != nil {
		return "", err
	}
	if len(resp.Kvs) == 0 {
		return "", fmt.Errorf("User %s not found in etcd", user)
	}
	return string(resp.Kvs[0].Value), nil

}

func (s *Store) GetGatewayForIP(ip net.IP) (*net.IPNet, *net.IP, error) {

	resp, err := s.kv.Get(context.TODO(), "/ipam/gateway/", clientv3.WithPrefix())
	if err != nil {
		return nil, nil, err
	}

	if len(resp.Kvs) == 0 {
		return nil, nil, fmt.Errorf("Gateway not found for %s", ip.String())
	}

	for _, item := range resp.Kvs {
		x := strings.Split(string(item.Value), ",")
		// subnet, err := types.ParseCIDR(strings.Split(string(item.Key)), "/")[]
		subnet, err := types.ParseCIDR(x[0])
		if err != nil {
			// TODO:
			continue
		}

		if subnet.Contains(ip) {
			// TODO: check for out of range
			gw := net.ParseIP(x[1])
			return subnet, &gw, nil

		}
	}
	return nil, nil, fmt.Errorf("Not subnet found for IP %s", ip.String())
}

func (s *Store) GetUsedByPod(pod string, namespace string) ([]net.IP, error) {
	resp, err := s.kv.Get(context.TODO(), "/ipam/ips/", clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}
	ret := make([]net.IP, 0)
	for _, item := range resp.Kvs {
		row := strings.Split(string(item.Value), ",")
		if row[2] == pod && row[3] == namespace {
			ret = append(ret, net.ParseIP(row[0]))

		}
	}
	return ret, nil
}

func (s *Store) Reserve(id string, ip net.IP, podName string, podNamespace string) (bool, error) {
	// TODO: lock
	if _, err := s.kv.Put(context.TODO(), "/ipam/ips/"+ip.String(),
		ip.String()+","+id+","+podName+","+podNamespace); err != nil {
		return false, nil
	}

	return true, nil
}

// LastReservedIP returns the last reserved IP if exists
func (s *Store) LastReservedIP(rangeID string) (net.IP, error) {
	resp, err := s.kv.Get(context.TODO(), "/ipam/last_reserved_ip"+rangeID)
	if err != nil {
		return nil, err
	}
	if len(resp.Kvs) == 0 {
		// case when initial state there is no this key.
		return nil, nil
	}
	return net.ParseIP(string(resp.Kvs[0].Value)), nil
}

func (s *Store) Release(ip net.IP) error {
	_, err := s.kv.Delete(context.TODO(), "/ipam/ips/"+ip.String())
	return err
}

// N.B. This function eats errors to be tolerant and
// release as much as possible
func (s *Store) ReleaseByID(id string) error {
	resp, err := s.kv.Get(context.TODO(), "/ipam/ips/", clientv3.WithPrefix())
	if err != nil {
		return err
	}

	if len(resp.Kvs) == 0 {
		// TODO: improve.
		return fmt.Errorf("No value in /ipam/ips")
	}

	for _, item := range resp.Kvs {
		if strings.TrimSpace(string(item.Value)) == strings.TrimSpace(id) {
			_, err = s.kv.Delete(context.TODO(), strings.TrimSpace(string(item.Key)))
			if err != nil {
				return err
			}
		}
	}
	return nil
}
