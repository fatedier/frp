FROM golang AS builder
WORKDIR /frp
COPY . .
RUN make

FROM golang
WORKDIR /
COPY --from=builder /frp/bin/frpc /usr/local/bin/frpc
COPY --from=builder /frp/conf/frpc.ini /etc/frpc.ini
CMD ["frpc", "--config", "/etc/frpc.ini"]

