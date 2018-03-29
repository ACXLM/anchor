# encoding=utf-8
from __future__ import print_function

import base64
import copy
import functools
import json
import os
import string
import ipaddress
import time
from collections import namedtuple
from datetime import datetime
from inspect import getargspec
import re

from errors import sip_ip_create_gateway_error

def str2bool(v):
    if v is None:
        return v
    return v.lower() in ('yes', 'true', 't', '1')


def convert_bool_args(args, bool_fields):
    args = copy.deepcopy(args)
    for f in bool_fields:
        if f in args:
            args[f] = str2bool(args[f])
    return args


def gen_random_str(length):
    return base64.b32encode(os.urandom(3 * length))[:length].lower()


def bytes_to_str(b):
    b = b[2:]
    n = 2
    b_a = [b[i:i + n] for i in range(0, len(b), n)]
    return "".join([chr(int(ib, 16)) for ib in b_a])


def print_dict(d, prefix=''):
    """
    :type d: dict
    """
    for k, v in d.items():
        if isinstance(v, dict):
            print('%s%s:' % (prefix, k))
            print_dict(v, prefix + ' ' * 4)
        else:
            print('%s%s = %s' % (prefix, k, v))


def memoize(fn):
    cache = fn.cache = {}

    @functools.wraps(fn)
    def _memoize(*args, **kwargs):
        kwargs.update(dict(zip(getargspec(fn).args, args)))
        key = tuple(kwargs.get(k, None) for k in getargspec(fn).args if k != 'self')
        if key not in cache:
            cache[key] = fn(**kwargs)
        return cache[key]

    return _memoize


def memoize_with_expire(expire):
    def _memoize(fn):
        cache = fn.cache = {}
        cache['__last_cached_time'] = time.time()

        @functools.wraps(fn)
        def __memoize(*args, **kwargs):
            kwargs.update(dict(zip(getargspec(fn).args, args)))
            key = tuple(kwargs.get(k, None) for k in getargspec(fn).args if k != 'self')
            if key not in cache or time.time() > cache['__last_cached_time'] + expire:
                cache[key] = fn(**kwargs)
                cache['__last_cached_time'] = time.time()
            return cache[key]

        return __memoize

    return _memoize


def memoize_in_request(fn):
    from flask import g

    @functools.wraps(fn)
    def _memoize(*args, **kwargs):
        kwargs.update(dict(zip(getargspec(fn).args, args)))
        key = '__cache__%s__' % fn.__name__ + ','.join(str(kwargs.get(k, None)) for k in getargspec(fn).args)
        if not hasattr(g, key):
            setattr(g, key, fn(**kwargs))
        return getattr(g, key)

    return _memoize


def memoize_in_object(fn):
    @functools.wraps(fn)
    def _memoize(self, *args, **kwargs):
        kwargs.update(dict(zip(getargspec(fn).args, args)))
        key = '__cache__%s__' % fn.__name__ + ','.join(str(kwargs.get(k, None)) for k in getargspec(fn).args)
        if not hasattr(self, key):
            setattr(self, key, fn(self, **kwargs))
        return getattr(self, key)

    return _memoize


def load_json_from(filename):
    with open(filename, 'r') as f:
        return json.load(f)


def dump_to(data, filename):
    with open(filename, 'w') as f:
        json.dump(data, f, indent=4)


class Template(string.Template):
    delimiter = '^^'


def load_template(filename):
    with open(filename, 'r') as f:
        s = f.read()
        return Template(s).substitute


def wrap_print(array, n, prefix='', sep=' '):
    maxlen = max([len(s) for s in array])
    fmt = '%-' + str(maxlen) + 's' + sep
    print(prefix, end='')
    for i, v in enumerate(array):
        if i % n == 0 and i:
            print('\n%s' % prefix, end='')
        print(fmt % v, end='')


def timestamp_to_iso(t):
    return datetime.fromtimestamp(int(t)).isoformat()


def exclude_methods(*methods):
    class ExcludeMethods(object):
        def __dir__(self):
            d = set(self.__dict__.keys() + dir(self.__class__))
            return [i for i in d if not i in methods]

        @property
        def __dict__(self):
            d = super(ExcludeMethods, self).__dict__
            return {k: v for k, v in d.items() if not k in methods}

        def __getattribute__(self, item):
            if item in methods:
                raise AttributeError("'%s' object has no attribute '%s'" % (self.__class__.__name__, item))
            return super(ExcludeMethods, self).__getattribute__(item)

    return ExcludeMethods


def methods_excluded(*methods):
    def decorator(base):
        class ExcludeMethods(base):
            def __dir__(self):
                d = set(self.__dict__.keys() + dir(self.__class__))
                return [i for i in d if not i in methods]

            @property
            def __dict__(self):
                d = super(ExcludeMethods, self).__dict__
                return {k: v for k, v in d.items() if not k in methods}

            def __getattribute__(self, item):
                if item in methods:
                    raise AttributeError("'%s' object has no attribute '%s'" % (self.__class__.__name__, item))
                return super(ExcludeMethods, self).__getattribute__(item)

        return ExcludeMethods

    return decorator


def enum(name, field_list_or_map):
    """
    :type name: str
    :type field_list_or_map: list<str> or map<str><str>
    :rtype: Union[Type[itemgetter], str, Type[OrderedDict], Type[property], Type[tuple]]
    """
    if isinstance(field_list_or_map, (list, tuple, set)):
        return namedtuple(name, field_list_or_map)(*range(len(field_list_or_map)))
    if isinstance(field_list_or_map, dict):
        return namedtuple(name, field_list_or_map.keys())(field_list_or_map.values())
    raise TypeError('field_list_or_map should be list<str> or map<str><str>')


def average(iterable, key=None):
    if key and callable(key):
        return sum(map(key, iterable)) / float(len(iterable))
    return sum(iterable) / float(len(iterable))


UNITS = ((1000, 'ns'), (1000, 'us'), (1000, 'ms'), (60, 's'), (60, 'm'), (24, 'h'), (None, 'd'))


def period_to_str(p):
    """
    >>> assert period_to_str(2*24*60*60*1000*1000*1000) == '2d'
    """
    if p is None:
        return
    p = float(p)
    u = ''
    for system, unit in UNITS:
        if system is None or p / system <= 1:
            return str(int(p)) + unit
        p = p / system
        u = unit
    return str(int(p)) + u


def str_to_period(s):
    """
    >>> assert str_to_period('2d') == 2*24*60*60*1000*1000*1000
    """
    if s is None:
        return
    s = s.lower()
    p = 0
    u = ''
    for i, c in enumerate(s):
        if not c.isdigit():
            p = int(s[:i])
            u = s[i:]
            break

    for system, unit in UNITS:
        if u == unit:
            return p
        p = p * system

    raise ValueError('Unknown unit')


@memoize
def current_plugin_token_name():
    import jwt

    token = os.getenv('DCE_PLUGIN_STORAGE_URL', '').split('//')[-1].split(':')[0]
    if not token:
        raise RuntimeError("No environment variable 'DCE_PLUGIN_STORAGE_URL'")
    name = jwt.decode(token, verify=False).get('plugin_name')
    return token, name

def underscore(word):
    word = re.sub(r"([A-Z]+)([A-Z][a-z])", r'\1_\2', word)
    word = re.sub(r"([a-z\d])([A-Z])", r'\1_\2', word)
    word = word.replace("-", "_")
    return word.lower()

def from_view_dict(values, with_order=False):
    if with_order:
        return dict((underscore(k), v) for k, v in values.items())
    return {underscore(k): v for k, v in values.items()}

def camelize(string, uppercase_first_letter=True):
    if uppercase_first_letter:
        return re.sub(r"(?:^|_)(.)", lambda m: m.group(1).upper(), string)
    else:
        return string[0].lower() + camelize(string)[1:]


def to_view_dict(values, with_order=False):
    if with_order:
        return dict((camelize(k), v) for k, v in values.items())
    return {camelize(k): v for k, v in values.items()}


def verify_ip(ip):
    try:
        ip = ipaddress.ip_address(ip)
        return ip
    except ValueError as e:
        return sip_ip_create_gateway_error(ip)


def verify_subnet(subnet):
    try:
        subnet = ipaddress.ip_network(subnet)
        return subnet
    except ValueError as e:
        return sip_ip_create_gateway_error(subnet)

def string_reverse(s):
    return s[::-1]
