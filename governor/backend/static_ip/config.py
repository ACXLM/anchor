import os
from functions import str2bool

DOCKER_HOST = os.getenv('DOCKER_HOST')
PROD = str2bool(os.getenv('PROD'))

ETCD_PORT =os.getenv('ETCD_PORT','12379')
ETCD_STATICIP_STORE_IP_PREFIX='/anchor/ips/'
ETCD_STATICIP_STORE_TENANT_PREFIX='/anchor/user/'
ETCD_STATICIP_STORE_GATEWAY_PREFIX = '/anchor/gw/'

CLIENT_CERTIFICATION_PATH = os.getenv('CLIENT_CERTIFICATION_PATH', '/etc/ssl/etcd/private/peer-cert.pem')
CLIENT_PRIVATE_KEY_PATH = os.getenv('CLIENT_PRIVATE_KEY_PATH', '/etc/ssl/etcd/private/peer-key.pem')
CLIENT_CERTIFICATION_CA_PATH = os.getenv('CLIENT_CERTIFICATION_CA_PATH', '/etc/ssl/etcd/private/ca.pem')

