import etcd3
import logging

from config import CLIENT_CERTIFICATION_CA_PATH
from config import CLIENT_CERTIFICATION_PATH
from config import CLIENT_PRIVATE_KEY_PATH


log = logging.getLogger(__name__)

def get_etcd():
    from config import ETCD_PORT
    from dce_plugin import PluginSDK
    ETCD_URL = PluginSDK()._detect_host_ip()
    host, port = ETCD_URL, ETCD_PORT
    try:
        etcd = etcd3.client(host=host, port=int(port),\
                            ca_cert=CLIENT_CERTIFICATION_CA_PATH, cert_key=CLIENT_PRIVATE_KEY_PATH, cert_cert=CLIENT_CERTIFICATION_PATH)
        etcd.put('/key', 'value')
    except etcd3.exceptions.ConnectionFailedError as e:
        log.error('Can not connect to {}:{},Message:{}'.format(host,port,e))
    else:
        return etcd

