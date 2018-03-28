import json
import logging
from flask import Response
from flask import jsonify
from flask import make_response
from flask import g
from flask_restful import Api, Resource, reqparse
from werkzeug.exceptions import HTTPException

from dce_client import require_admin
from dce_client import require_auth
from dce_client import get_tenant_names
from errors import APIException
from functions import convert_bool_args
from functions import from_view_dict
from job import get_static_ip_admin
from job import get_static_ip_unadmin
from job import create_bulk_static_ip
from job import get_tenant_ip
from job import bulk_delete_sip
from job import get_gateway
from job import create_gateway
from job import delete_gateway


log = logging.getLogger(__name__)


def error_message(error_id='', message=None):
    return Response(json.dumps({'id': error_id, 'message': message or error_id}, indent=4), mimetype='application/json')


class SIPApi(Api):
    def handle_error(self, e):
        if isinstance(e, APIException):
            return error_message(e.error_id, e.message), e.code
        if isinstance(e, HTTPException):
            return error_message('unknown_exception', str(e)), 400
        log.error(str(e), exc_info=True)
        raise e


def load_api(app):
    api = SIPApi(app)
    api.add_resource(SIPBaseApi, '/api/v1/static_ip', methods=['GET'])
    api.add_resource(SIPTenantIp, '/api/v1/tenant_ip', methods=['GET', 'POST'])
    api.add_resource(SIPBulkDeleteApi, '/api/v1/bulk_delete_sip', methods=['POST'])
    api.add_resource(GetTenantName, '/api/v1/get_tenant_name', methods=['GET'])
    api.add_resource(SIPGeteway, '/api/v1/gateway', methods=['GET', 'POST', 'DELETE', 'OPTION'])


class GetTenantName(Resource):
    @require_auth
    def get(self):
        return jsonify(get_tenant_names())


class SIPBaseApi(Resource):
    @require_auth
    def get(self):
        if g.is_admin:
            return get_static_ip_admin()
        else:
            return get_static_ip_unadmin()


class SIPTenantIp(Resource):
    @require_auth
    def get(self):
        args = reqparse.RequestParser(). \
            add_argument('TenantName', required=True). \
            add_argument('All', required=False, default='0'). \
            add_argument('Unuse', required=False, default='0'). \
            parse_args()
        args = convert_bool_args(args, ('All',))
        args = convert_bool_args(args, ('Unuse',))
        args = from_view_dict(args)
        return get_tenant_ip(**args)

    @require_auth
    @require_admin
    def post(self):
        args = reqparse.RequestParser(). \
            add_argument('StartIp', required=True). \
            add_argument('EndIp', required=True). \
            add_argument('TenantName', required=True). \
            parse_args()
        args = from_view_dict(args)
        return make_response(create_bulk_static_ip(**args), 201)


class SIPBulkDeleteApi(Resource):
    @require_auth
    @require_admin
    def post(self):
        args = reqparse.RequestParser(). \
            add_argument('StaticIps', required=True, action='append'). \
            add_argument('TenantName', required=True). \
            parse_args()
        args = from_view_dict(args)
        return make_response(bulk_delete_sip(**args), 204)


class SIPGeteway(Resource):
    @require_auth
    @require_admin
    def get(self):
        return get_gateway()


    @require_auth
    @require_admin
    def post(self):
        args = reqparse.RequestParser(). \
            add_argument('Subnet', required=True). \
            add_argument('Gateway', required=True). \
            parse_args()
        args = from_view_dict(args)
        return make_response(create_gateway(**args), 201)


    @require_auth
    @require_admin
    def delete(self):
        args = reqparse.RequestParser(). \
            add_argument('Subnet', required=True). \
            parse_args()
        args = from_view_dict(args)
        log.debug('delete dateway {}'.format(args))
        return make_response(delete_gateway(**args), 204)

