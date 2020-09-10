FROM golang:alpine

COPY . /go/src/github.com/fatedier/frp

RUN apk update \
 && apk add make \
 && cd /go/src/github.com/fatedier/frp \
 && make \
 && mv bin/frps /frps \
 && mv conf/frps.ini /frps.ini \
 && make clean \
 && rm -r /go

WORKDIR /

EXPOSE 80 443 6000 7000 7500

ENTRYPOINT ["/frps"]
