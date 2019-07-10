FROM golang AS builder
WORKDIR /frp
COPY . .
RUN make

FROM golang
WORKDIR /
COPY --from=builder /frp/bin/frps /usr/local/bin/frps
COPY --from=builder /frp/conf/frps.ini /etc/frps.ini
EXPOSE 7000 7500
CMD ["frps", "--config", "/etc/frps.ini"]
