import json
import requests
from flask import g, request
from functools import wraps
from requests.auth import HTTPBasicAuth

from requests.packages.urllib3.exceptions import InsecureRequestWarning

requests.packages.urllib3.disable_warnings(InsecureRequestWarning)


class DCEClient(requests.Session):
    @staticmethod
    def dce_host():
        import dce_plugin
        sdk = dce_plugin.PluginSDK()
        return 'https://{}:{}'.format(sdk._detect_host_ip(),
                                      sdk._detect_dce_ports()[1])

    def __init__(self, base_url=None, username=None, password=None, token=None, timeout=30, verify=False):
        super(DCEClient, self).__init__()
        self.base_url = base_url or self.dce_host()
        self.timeout = timeout
        self.verify = verify
        self.username = username
        self.password = password
        self.token = request.headers.get('X-DCE-Access-Token') \
                     or request.cookies.get('DCE_TOKEN')
        if self.username and self.password:
            self.auth = HTTPBasicAuth(self.username, self.password)
        if self.token:
            self.headers['X-DCE-Access-Token'] = self.token
        self.headers['User-Agent'] = 'DCE-plugin/static-ip'

    def _url(self, path):
        return '{0}{1}'.format(self.base_url, path)

    def _get(self, url, **kwargs):
        return self.get(url, **kwargs)

    def _post(self, url, **kwargs):
        return self.post(url, **kwargs)

    def _post_json(self, url, data, **kwargs):
        if 'headers' not in kwargs:
            kwargs['headers'] = {}
        kwargs['headers']['Content-Type'] = 'application/json'
        return self._post(url, data=json.dumps(data), **kwargs)

    def _raise_for_status(self, response):
        try:
            response.raise_for_status()
        except requests.exceptions.HTTPError as e:
            raise e

    def _result(self, response, is_json=True):
        self._raise_for_status(response)
        if is_json:
            return response.json()
        return response.text

    def get_account(self):
        path = '/dce/my-account'
        try:
            res = self._result(self._get(self._url(path)))
            return res
        except Exception:
            return {}

    def is_admin(self):
        path = '/dce/my-account'
        try:
            res = self._result(self._get(self._url(path)))
        except Exception:
            return False
        return res.get('IsAdmin', False)

    def tenants(self):
        path = '/dce/tenants'
        try:
            return self._result(self._get(self._url(path)))
        except Exception as e:
            return []


def require_auth(f):
    @wraps(f)
    def decorated(*args, **kwargs):
        c = DCEClient()
        account = c.get_account()
        g.username = account.get('Name', '')
        g.is_admin = account.get('IsAdmin', False)
        return f(*args, **kwargs)
    return decorated


def require_admin(f):
    @wraps(f)
    def decorated(*args, **kwargs):
        if not g.is_admin:
            return '', 401
        return f(*args, **kwargs)
    return decorated


def get_tenant_names():
    return [i.get('Name') for i in DCEClient().tenants()]


if __name__ == "__main__":
    c = DCEClient(base_url="https://192.168.4.216:443", username="admin", password="admin")
#    print(get_tenant_names())
#    print(c.is_admin())
#    get_service_name('dao-2048qwertyuiop')

