#!/bin/sh

cat <<EOF > /etc/supervisord.conf
[supervisord]
nodaemon=true

[program:nginx]
command=nginx -g "daemon off;" -c /etc/nginx/static-ip/server-nginx.conf
autorestart=true
startsecs=0
stderr_logfile=/dev/stderr
stderr_logfile_maxbytes=0
stdout_logfile=/dev/stdout
stdout_logfile_maxbytes=0

[program:static-ip-server]
command=/usr/local/bin/python /usr/src/app/static-ip/app.py
autorestart=true
startsecs=0
stderr_logfile=/dev/stderr
stderr_logfile_maxbytes=0
stdout_logfile=/dev/stdout
stdout_logfile_maxbytes=0
EOF

exec /usr/bin/supervisord
