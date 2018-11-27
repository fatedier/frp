FROM golang:1.10 AS base

ENV CGO_ENABLED=0 
ENV GOOS=linux
ENV GOARCH=amd64
ENV SRCPATH=/go/src/github.com/fatedier/frp

RUN go get github.com/fatedier/frp || true  && \
        cd $SRCPATH \
        && make

FROM scratch
ENV SRCPATH=/go/src/github.com/fatedier/frp
COPY --from=base $SRCPATH/bin/frpc /frpc 
COPY --from=base $SRCPATH/bin/frps /frps 
COPY --from=base $SRCPATH/conf/frpc.ini /frpc.ini 
COPY --from=base $SRCPATH/conf/frps.ini /frps.ini 

WORKDIR /

EXPOSE 80 443 6000 7000 7500

ENTRYPOINT ["/frps"]
