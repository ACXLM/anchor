#!/usr/bin/env python
#coding=utf-8

import logging
from flask import jsonify

from dce_client import get_tenant_names
from functions import to_view_dict
from functions import verify_ip
from functions import verify_subnet
from errors import sip_already_exist_error
from errors import sip_not_exist_error
from errors import sip_ip_format_error
from errors import sip_ip_range_too_big
from errors import sip_ip_range_err
from errors import sip_not_belong_to_tenant
from errors import sip_ip_gateway_not_in_subnet_error
from errors import sip_ip_delete_gateway_error
from errors import subnet_already_exist_error
from pluginetcd import get_etcd
from config import ETCD_STATICIP_STORE_IP_PREFIX
from config import ETCD_STATICIP_STORE_TENANT_PREFIX
from config import ETCD_STATICIP_STORE_GATEWAY_PREFIX


log = logging.getLogger(__name__)
etcd = get_etcd()


class StaticIP(object):
    def __init__(self, static_ip, pod_name, tenant_name='', service_name='', app_name=''):
        self.static_ip = static_ip
        self.pod_name = pod_name
        self.tenant_name = tenant_name
        self.app_name = app_name
        self.service_name = service_name


    def _as_dict(self):
        d = dict(
            static_ip=self.static_ip,
            tenant_name=self.tenant_name,
            app_name=self.app_name,
            service_name=self.service_name,
            pod_name=self.pod_name
        )

        return d

    def _as_view_dict(self):
        return to_view_dict(self._as_dict())


class Job(object):
    def __init__(self, tenant_name):
        self.tenant_name = tenant_name
        self.static_tenant_ips_all = etcd.get("{}{}".format(ETCD_STATICIP_STORE_TENANT_PREFIX, self.tenant_name))[0]

    def get_tenant_ip_ips_use(self):
        l = []
        for s in etcd.get_prefix(ETCD_STATICIP_STORE_IP_PREFIX):
            sip = s[0].split(",")
            if sip[2] == self.tenant_name:
                l.append(sip[0])
        return l

    def get_static_ips_all(self):
        sip = ""
        for s in etcd.get_prefix(ETCD_STATICIP_STORE_TENANT_PREFIX):
            sip = sip + ',' + s[0]
        log.debug('get_static_ips_all:{}'.format(sip))
        return sip.split(",")

    @staticmethod
    def ips(start_ip, end_ip):
        import socket
        import struct

        try:
            start_s = socket.inet_aton(start_ip)
            end_s = socket.inet_aton(end_ip)
        except Exception as e:
            log.error(e)
            return sip_ip_format_error()

        start = struct.unpack('>I', start_s)[0]
        end = struct.unpack('>I', end_s)[0]
        return [socket.inet_ntoa(struct.pack('>I', i)) for i in range(start, end + 1)]

    @staticmethod
    def verify_ips_not_exist(self, start_ip, end_ip):
        ips = self.ips(start_ip, end_ip)
        if len(self.get_static_ips_all()) == 0:
            return True
        else:
            exist_ips = self.get_static_ips_all()
        dup = list(set(exist_ips).intersection(set(ips)))
        if any(dup):
            return False
        return True

    @staticmethod
    def verify_ips_exist(self, static_ips):
        static_ips = static_ips
        log.debug("verify_ips_exist:{}".format(self.get_static_ips_all()))
        if any(self.get_static_ips_all()):
            exist_ips = self.get_static_ips_all()
        else:
            return False
        dup = list(set(exist_ips).intersection(set(static_ips)))
        log.debug("exist_ips:{}".format(exist_ips))
        log.debug("start_ip:{}".format(static_ips))
        log.debug("dup:{}".format(dup))
        if any(dup):
            return True
        return False

    #todo return
    def bulk_create_static_ip(self, start_ip, end_ip):
        store_ip = ""
        if not self.verify_ips_not_exist(self, start_ip, end_ip):
            return sip_already_exist_error()

        start_ip_s = start_ip.split('.')
        end_ip_s = end_ip.split('.')
        if start_ip_s[0] != end_ip_s[0] or start_ip_s[1] != end_ip_s[1] or start_ip_s[2] != end_ip_s[2]:
            return sip_ip_range_too_big()

        if int(start_ip_s[3]) > int(end_ip_s[3]):
            return sip_ip_range_err()

        if self.static_tenant_ips_all != None:
            tenant_exist_ip = self.static_tenant_ips_all.split(',')
            for i in tenant_exist_ip:
                store_ip = store_ip + i + ","
            for j in self.ips(start_ip, end_ip):
                store_ip = store_ip + j + ","
            store_ip = store_ip[:-1]
            with etcd.lock('add') as lock:
                etcd.put('{}{}'.format(ETCD_STATICIP_STORE_TENANT_PREFIX, self.tenant_name), store_ip)
            return jsonify({"Status":"Ok"})
        else:
            for i in self.ips(start_ip, end_ip):
                store_ip = store_ip + i + ","
            store_ip = store_ip[:-1]
            with etcd.lock('create') as lock:
                etcd.put('{}{}'.format(ETCD_STATICIP_STORE_TENANT_PREFIX, self.tenant_name), store_ip)
            return jsonify({"Status":"Ok"})

    #todo return
    def bulk_delete_static_ip(self, static_ips):
        store_ip = ""
        if not self.verify_ips_exist(self, static_ips):
            return sip_not_exist_error()

        if self.static_tenant_ips_all != None:
            if not any([ip for ip in static_ips if ip in self.static_tenant_ips_all.split(',')]):
               return sip_not_belong_to_tenant(self.tenant_name)

            ips = list(set(self.static_tenant_ips_all.split(',')).\
                                  difference(set(static_ips)))
            log.debug('from bulk_delete_static_ip ips: {}'.format(ips))
            if any(ips):
                for i in ips:
                    store_ip = store_ip + i + ","
                store_ip = store_ip[:-1]
                log.debug('from bulk_delete_static_ip store_ip: {}'.format(store_ip))
                with etcd.lock('delete') as lock:
                    etcd.put('{}{}'.format(ETCD_STATICIP_STORE_TENANT_PREFIX, self.tenant_name), store_ip)
                return jsonify({"Status": "Ok"})
            else:
                etcd.delete('{}{}'.format(ETCD_STATICIP_STORE_TENANT_PREFIX,self.tenant_name))
                return jsonify({"Status":"Ok"})
        else:
            return sip_not_exist_error()


def get_static_ip_admin():
    result = {}
    sips = etcd.get_prefix(ETCD_STATICIP_STORE_IP_PREFIX)
    log.debug("sip from admin:{}".format(sips))
    for s in sips:
        data = s[0].split(",")
        if len(data) != 5:
            continue
        result[data[0]] = StaticIP(static_ip=data[0], pod_name=data[1],
                                   tenant_name=data[2], app_name=data[3],
                                   service_name=data[4])._as_view_dict()
    log.debug('get_static_ip_admin: {}'.format(result))
    return jsonify(result)


def get_static_ip_unadmin():
    result = {}
    sips = etcd.get_prefix(ETCD_STATICIP_STORE_IP_PREFIX)
    log.debug("sip from unadmin:{}".format(sips))
    for s in sips:
        data = s[0].split(",")
        if len(data) != 5:
            continue
        if data[2] in get_tenant_names():
            result[data[0]] = StaticIP(static_ip=data[0], pod_name=data[1],
                                           tenant_name=data[2],app_name=data[3],
                                           service_name=data[4])._as_view_dict()
    log.debug('get_static_ip_unadmin: {}'.format(result))
    return jsonify(result)


def get_tenant_ip(tenant_name, all=False, unuse=False):
    j = Job(tenant_name)
    if unuse:
        return jsonify({tenant_name:\
                        list(set(j.static_tenant_ips_all.split(","))\
                             .difference(set(j.get_tenant_ip_ips_use())))\
                        if j.static_tenant_ips_all else sip_not_exist_error(tenant_name)})
    else:
        if all:
            return jsonify({tenant_name:j.static_tenant_ips_all.split(",")})\
                if j.static_tenant_ips_all else sip_not_exist_error(tenant_name)
        else:
            return jsonify({tenant_name:j.get_tenant_ip_ips_use()})


def create_bulk_static_ip(start_ip, end_ip, tenant_name):
    j = Job(tenant_name=tenant_name)
    return j.bulk_create_static_ip(start_ip=start_ip, end_ip=end_ip)


def bulk_delete_sip(static_ips, tenant_name):
    j = Job(tenant_name=tenant_name)
    return j.bulk_delete_static_ip(static_ips)


def get_gateway():
    l = []
    gateway = etcd.get_prefix('{}'.format(ETCD_STATICIP_STORE_GATEWAY_PREFIX))
    for i in gateway:
        d = {}
        gw = i[0]
        if any(gw):
            key = gw.split(',')[0]
            value = gw.split(',')[1]
            d['Subnet'] = key
            d['Gateway'] = value
            l.append(d)
    return jsonify(l)


def create_gateway(subnet, gateway):
    d = {}
    subnet = verify_subnet(subnet)
    gateway = verify_ip(gateway)
    exist_subnet = get_subnet()
    if str(subnet) in exist_subnet:
        return subnet_already_exist_error(str(subnet))
    if gateway in subnet:
        with etcd.lock('create_gateway') as lock:
            etcd.put('{}{}'.format(ETCD_STATICIP_STORE_GATEWAY_PREFIX, str(subnet)),\
                         '{},{}'.format(str(subnet), str(gateway)))
        d['Gateway'] = str(gateway)
        d['Subnet'] = str(subnet)
        return jsonify(d)
    else:
        return sip_ip_gateway_not_in_subnet_error(str(subnet))


def delete_gateway(subnet):
    exist_subnet = get_subnet()
    log.debug('exist_subnet {}'.format(exist_subnet))
    if len(exist_subnet) == 0:
        log.debug('error in len')
        return sip_ip_delete_gateway_error(subnet)
    elif subnet not in exist_subnet:
        log.debug('error in sub subnet:{}'.format(subnet))
        return sip_ip_delete_gateway_error(subnet)
    else:
        etcd.delete('{}{}'.format(ETCD_STATICIP_STORE_GATEWAY_PREFIX,subnet))
        return jsonify({'Subnet':subnet})


def get_subnet():
    l = []
    subnet = etcd.get_prefix('{}'.format(ETCD_STATICIP_STORE_GATEWAY_PREFIX))
    for i in subnet:
        l.append(i[0].split(',')[0])
    return l
