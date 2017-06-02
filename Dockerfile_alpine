FROM alpine:3.5

COPY tmp/frpc /frpc
COPY tmp/frps /frps
COPY conf/frpc_min.ini /frpc.ini
COPY conf/frps_min.ini /frps.ini

WORKDIR /

EXPOSE 80 443 6000 7000 7500

ENTRYPOINT ["/frps"]
