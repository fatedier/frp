FROM golang:1.6

RUN go get github.com/tools/godep
COPY . /go/src/github.com/fatedier/frp
RUN cd /go/src/github.com/fatedier/frp   \
 && make                                 \
 && mv bin/frpc bin/frps /usr/local/bin  \
 && mv conf/*.ini /
WORKDIR /
ENTRYPOINT ["frps"]
EXPOSE 6000 7000 7500
