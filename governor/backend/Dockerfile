FROM python:2.7-alpine

LABEL maintainer="hui.zhu" \
      io.daocloud.dce.plugin.name="Anchor" \
      io.daocloud.dce.plugin.description="统一管理和分配静态IP地址，每个容器都可以拥有独立的静态IP。" \
      io.daocloud.dce.plugin.categories="container-tool" \
      io.daocloud.dce.plugin.vendor="DaoCloud" \
      io.daocloud.dce.plugin.required-dce-version=">=2.10.0" \
      io.daocloud.dce.plugin.nano-cpus-limit="1000000000" \
      io.daocloud.dce.plugin.memory-bytes-limit="2097152000"

WORKDIR /usr/src/app/


COPY ./requirements.txt /usr/src/app/
RUN apk add --no-cache libstdc++ \
  && apk add --no-cache --virtual .build-deps \
    python-dev \
    build-base \
    linux-headers \
  && pip install --no-cache-dir -r /usr/src/app/requirements.txt \
  && apk del .build-deps \
  && apk --update add nginx supervisor \
  && rm -rf /usr/src/python ~/.cache \
  && rm -rf /var/cache/apk/*

ENV PROD True

EXPOSE 80

COPY ./certs /etc/ssl
COPY ./plugin.json /usr/src/app/plugin.json
COPY ./nginx /etc/nginx/static-ip
COPY ./bin /static-ip

COPY ./ui /usr/src/app/ui
COPY ./static_ip /usr/src/app/static-ip

CMD ["/static-ip/server.sh"]
