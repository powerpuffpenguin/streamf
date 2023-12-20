FROM alpine:3.17.1

RUN set -eux;  \
    apk add --no-cache tzdata

Copy root /

RUN set -eux;  \
    chown root.root  /usr/local/bin/docker-entrypoint.sh; \
    chown root.root  /usr/local/bin/streamf; \
    chmod a+x /usr/local/bin/docker-entrypoint.sh;

WORKDIR /
ENTRYPOINT ["docker-entrypoint.sh"]
VOLUME /data

CMD ["default-command"]