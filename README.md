# Anchor: Static IP CNI solution for Kubernetes

Anchor is a static IP CNI solution for kubernetes. It is composed of anchor-ipam, anchor-governor.

## Create Network interface

* MacVLAN

The code below create a macvlan interface named mac0, and its parent is eth0.

 ```shell
ip link add mac0 link eth0 type macvlan mode bridge
ip addr add 10.100.160.120/24 dev mac0
ip link set dev mac0 up

ip route flush dev mac0
ip route add 10.100.160.0/24 dev mac0 metric 0
```

* Bridge

Bridge with promiscuous mode is supported. The method will be added here later.

# Anchor governor

Anchor governor is the manager of the etcd store. It is responsible for init the `User <-> IPs`, and display the usage of the IPs.

# ETCD

Recently, please use etcd only as data store. Please intall or ensure that there is an etcd cluster available first. We used it as a distributed database.

## Configuration and installation

```
vi anchor-ipam/k8s-install/anchor-with(or without)-rbac.yaml
# Config line 9 and line 38, 39, 40
```

```shell
kubectl apply -f anchor-ipam/k8s-install/anchor-with(or without)-rbac.yaml
```

## Init and Example

```shell
etcdctl put /ipam/users/user01 /ipam/users/user01,192.168.2.[2-19]
etcdctl put /ipam/gateway/192.168.2.0/16 192.168.2.0/16,192.168.2.1
```

Of course, we can init the database use anchor govenor.

example.yaml

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: busybox
  namespace: default
  annotations:
    cni.daocloud.io/ipAddrs: 192.168.2.[2-8]
    cni.daocloud.io/currentUser: user01
spec:
  containers:
  - image: busybox
    command:
      - sleep
      - "3600"
    imagePullPolicy: IfNotPresent
    name: busybox
  restartPolicy: Never
```

```shell
kubectl apply -f example.yaml
```
