FROM alpine:latest

ENV VERSION=0.33.0
ENV FILENAME=frp_${VERSION}_linux_amd64

RUN set -ex; \
    \
    cd /tmp; \
    wget https://github.com/fatedier/frp/releases/download/v${VERSION}/${FILENAME}.tar.gz; \
    tar xf ${FILENAME}.tar.gz; \
    mv ${FILENAME}/frps /usr/local/bin; \
    mv ${FILENAME}/frps.ini /etc/frps.ini

WORKDIR /

ENTRYPOINT ["frps"]

CMD [ "-c", "/etc/frps.ini" ]
