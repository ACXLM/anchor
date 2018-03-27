# host-etcd IP address management plugin

host-etcd IPAM allocates IPv4 and IPv6 addresses out of a specified address range. Optionally,
it can include a DNS configuration from a `resolv.conf` file on the host.

## Overview

host-etcd IPAM plugin allocates ip addresses out of a set of address ranges.
It stores the state on a etcd store. therefore ensuring uniqueness of IP addresses over the cluster.

The allocator can allocate multiple ranges, and supports sets of multiple (disjoint) 
subnets. The allocation strategy is loosely round-robin within each range set.

## Example configurations

Note that the key `ranges` is a list of range sets. That is to say, the length 
of the top-level array is the number of addresses returned. The second-level 
array is a set of subnets to use as a pool of possible addresses.

This example configuration returns 2 IP addresses.

```json
{
	"ipam": {
		"type": "host-etcd",
		"ranges": [
			[
				{
					"subnet": "10.10.0.0/16",
					"rangeStart": "10.10.1.20",
					"rangeEnd": "10.10.3.50",
					"gateway": "10.10.0.254"
				},
				{
					"subnet": "172.16.5.0/24"
				}
			],
			[
				{
					"subnet": "3ffe:ffff:0:01ff::/64",
					"rangeStart": "3ffe:ffff:0:01ff::0010",
					"rangeEnd": "3ffe:ffff:0:01ff::0020"
				}
			]
		],
		"routes": [
			{ "dst": "0.0.0.0/0" },
			{ "dst": "192.168.0.0/16", "gw": "10.10.5.1" },
			{ "dst": "3ffe:ffff:0:01ff::1/64" }
		],
		"endpoints": ["127.0.0.1:2379"]
	}
}
```

We can test it out on the command-line:

```bash
$ echo '{ "cniVersion": "0.3.1", "name": "examplenet", "ipam": { "type": "host-etcd", "ranges": [ [{"subnet": "203.0.113.0/24"}], [{"subnet": "2001:db8:1::/64"}]], "endpoints": ["127.0.0.1"]  } }' | CNI_COMMAND=ADD CNI_CONTAINERID=example CNI_NETNS=/dev/null CNI_IFNAME=dummy0 CNI_PATH=. ./host-etcd

```

```json
{
    "ips": [
        {
            "version": "4",
            "address": "203.0.113.2/24",
            "gateway": "203.0.113.1"
        },
        {
            "version": "6",
            "address": "2001:db8:1::2/64",
            "gateway": "2001:db8:1::1"
        }
    ],
    "dns": {}
}
```

## Network configuration reference

* `type` (string, required): "host-etcd".
* `routes` (string, optional): list of routes to add to the container namespace. Each route is a dictionary with "dst" and optional "gw" fields. If "gw" is omitted, value of "gateway" will be used.
* `resolvConf` (string, optional): Path to a `resolv.conf` on the host to parse and return as the DNS configuration
* `endpoints` ([]string, required): Endpoints of the etcd store use for maintaining state, e.g. which IPs have been allocated to which containers
* `ranges`, (array, required, nonempty) an array of arrays of range objects:
	* `subnet` (string, required): CIDR block to allocate out of.
	* `rangeStart` (string, optional): IP inside of "subnet" from which to start allocating addresses. Defaults to ".2" IP inside of the "subnet" block.
	* `rangeEnd` (string, optional): IP inside of "subnet" with which to end allocating addresses. Defaults to ".254" IP inside of the "subnet" block for ipv4, ".255" for IPv6
	* `gateway` (string, optional): IP inside of "subnet" to designate as the gateway. Defaults to ".1" IP inside of the "subnet" block.

## Supported arguments
The following [CNI_ARGS](https://github.com/containernetworking/cni/blob/master/SPEC.md#parameters) are supported:

* `ip`: request a specific IP address from a subnet.

The following [args conventions](https://github.com/containernetworking/cni/blob/master/CONVENTIONS.md) are supported:

* `ips` (array of strings): A list of custom IPs to attempt to allocate

### Custom IP allocation
For every requested custom IP, the `host-etcd` allocator will request that IP
if it falls within one of the `range` objects. Thus it is possible to specify
multiple custom IPs and multiple ranges.

If any requested IPs cannot be reserved, either because they are already in use
or are not part of a specified range, the plugin will return an error.


## Etcd store

Allocated IP addresses are stored as kv pairs in `/ipam/ips/$NETWORK_NAME`.
The path will be customized later.

## TODO

Package k8s should be rename to runtime and k8s is an implement of runtime.

An interface of `Runtime` is needed.
Deletor
