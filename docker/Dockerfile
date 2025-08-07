FROM alpine:3.22

RUN set -eux;  \
    apk add --no-cache tzdata

Copy root /

RUN set -eux;  \
    chown root:root  /usr/local/bin/docker-entrypoint.sh; \
    chown root:root  /usr/local/bin/streamf; \
    chown root:root  /usr/local/bin/docker-entrypoint.sh; \
    chmod a+x /usr/local/bin/docker-entrypoint.sh;

WORKDIR /
ENTRYPOINT ["docker-entrypoint.sh"]
VOLUME /data

CMD ["default-command"]