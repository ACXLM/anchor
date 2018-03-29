import os
import sys
import time
import schedule
import threading
import logging
from flask import Flask
from flask import request
from flask import g
from flask_cors import CORS

from api import load_api
from config import PROD


ROOT_PATH = os.path.abspath(os.path.dirname(__file__))


def setup_app(app):
    @app.before_request
    def get_info_from_header():
        g.token = request.headers.get('X-DCE-Access-Token') or request.cookies.get('DCE_TOKEN')
        g.username = request.headers.get('X-DCE-Username') or None
        g.password = request.headers.get('X-DCE-Password') or None
#        g.tenant = request.headers.get('DCE-TENANT') or request.cookies.get('DCE_TENANT')


LOG_LEVEL = logging.INFO if PROD else logging.DEBUG
LOG_FORMAT = '%(asctime)s (%(process)d/%(threadName)s) %(name)s %(levelname)s - %(message)s'


def setup_logging(level=None):
    level = level or LOG_LEVEL
    console_handler = logging.StreamHandler(sys.stderr)
    formatter = logging.Formatter(LOG_FORMAT)
    console_handler.setFormatter(formatter)

    root_logger = logging.getLogger()
    root_logger.addHandler(console_handler)
    root_logger.setLevel(level)

    # setup logging level
    logging.getLogger('requests').propagate = False
    logging.getLogger('docker').level = logging.ERROR
    logging.getLogger('schedule').level = logging.WARN


def __start_schedule_deamon():
    def schedule_run():
        while True:
            schedule.run_pending()
            # TODO performance test
            time.sleep(5)

    t = threading.Thread(target=schedule_run)
    t.setDaemon(True)
    t.start()


def __init():
    setup_logging()

    __start_schedule_deamon()

__init()


def create_app(name=None):
    app = Flask(name or __name__)
    CORS(app)
    app.debug = False
    setup_app(app)
    load_api(app)

    return app


if __name__ == '__main__':
    app = create_app()
    app.run('0.0.0.0', port=8000, debug=True)
