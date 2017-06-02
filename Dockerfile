FROM golang:1.8

COPY . /go/src/github.com/fatedier/frp

RUN cd /go/src/github.com/fatedier/frp \
 && make \
 && mv bin/frpc /frpc \
 && mv bin/frps /frps \
 && mv conf/frpc_min.ini /frpc.ini \
 && mv conf/frps_min.ini /frps.ini \
 && make clean

WORKDIR /

EXPOSE 80 443 6000 7000 7500

ENTRYPOINT ["/frps"]
