FROM golang:1.10 as frpBuild

COPY . /go/src/github.com/fatedier/frp

ENV CGO_ENABLED=0

RUN cd /go/src/github.com/fatedier/frp \
 && make

FROM alpine:3.6

COPY --from=frpBuild /go/src/github.com/fatedier/frp/bin/frpc /
COPY --from=frpBuild /go/src/github.com/fatedier/frp/conf/frpc.ini /
COPY --from=frpBuild /go/src/github.com/fatedier/frp/bin/frps /
COPY --from=frpBuild /go/src/github.com/fatedier/frp/conf/frps.ini /

EXPOSE 80 443 6000 7000 7500

WORKDIR /

CMD ["/frps","-c","frps.ini"]