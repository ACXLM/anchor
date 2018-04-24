FROM alpine

LABEL maintainer "Haines Chan <yunhai.chen@daocloud.io>"

RUN apk --no-cache add iproute2

ADD anchor-ipam /opt/cni/bin/anchor-ipam
ADD k8s-install/install-cni.sh /install-cni.sh
ADD k8s-install/anchor.conf.default /calico.conf.tmp

ENV PATH=$PATH:/opt/cni/bin
VOLUME /opt/cni
WORKDIR /opt/cni/bin
CMD ["/opt/cni/bin/anchor-ipam"]
