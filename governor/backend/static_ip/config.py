import os
from functions import str2bool

DOCKER_HOST = os.getenv('DOCKER_HOST')
PROD = str2bool(os.getenv('PROD'))
GUNICORN_WORKERS = os.getenv('GUNICORN_WORKERS')

ETCD_PORT =os.getenv('ETCD_PORT','12379')
ETCD_STATICIP_STORE_IP_PREFIX='/DCEPlugin/Anchor/ips/'
ETCD_STATICIP_STORE_TENANT_PREFIX='/DCEPlugin/Anchor/Tenant/'
ETCD_STATICIP_STORE_GATEWAY_PREFIX = '/DCEPlugin/Anchor/Gateway/'

CLIENT_CERTIFICATION_PATH = os.getenv('CLIENT_CERTIFICATION_PATH', '/etc/ssl/private/peer-cert.pem')
CLIENT_PRIVATE_KEY_PATH = os.getenv('CLIENT_PRIVATE_KEY_PATH', '/etc/ssl/private/peer-key.pem')
CLIENT_CERTIFICATION_CA_PATH = os.getenv('CLIENT_CERTIFICATION_CA_PATH', '/etc/ssl/private/ca.pem')

#test
#ETCD_URL="192.168.4.216:12379,192.168.4.215:12379,192.168.4.156:12379"
#ETCD_URL = os.getenv('ETCD_URL')

