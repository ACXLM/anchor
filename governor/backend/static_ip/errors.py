def self_service_not_found():
    raise Exception('self_service_not_found')


class APIException(Exception):
    def __init__(self, error_id, message, code=500):
        super(APIException, self).__init__(message)
        self.message = message
        self.error_id = error_id
        self.code = code


def sip_already_exist_error(message=''):
    raise APIException('static_ip_already_exist_error', message, 400)


def sip_not_exist_error(message=''):
    raise APIException('static_ip_not_exist_error', message, 400)


def not_authorized_error(message):
    raise APIException('not_authorized_error', message, 401)


def sip_already_assigned_error(message=''):
    raise APIException('static_ip_already_assigned_error', message, 400)


def sip_delete_error(message=''):
    raise APIException('static_ip_delete_error', message, 400)


def sip_ip_format_error(message=''):
    raise APIException('static_ip_format_error', message, 400)


def sip_ip_range_too_big(message=''):
    raise APIException('static_ip_range_too_big', message, 400)


def sip_unassign_to_service_first(message=''):
    raise APIException('static_ip_unassign_to_service_first', message, 400)


def sip_not_belong_to_tenant(message=''):
    raise APIException('static_ip_not_belong_to_tenant', message, 400)

def sip_ip_range_err(message=''):
    raise APIException('static_ip_range_err', message, 400)


def sip_ip_gateway_not_in_subnet_error(message=''):
    raise APIException('sip_ip_gateway_not_in_subnet_error', message, 400)


def sip_ip_delete_gateway_error(message=''):
    raise APIException('sip_ip_gateway_not_exist_error', message, 400)

def sip_ip_create_gateway_error(message=''):
    raise APIException('sip_ip_gateway_format_error', message, 400)

def subnet_already_exist_error(message=''):
    raise APIException('subnet_already_exist_error', message, 400)

