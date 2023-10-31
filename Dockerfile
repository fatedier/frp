FROM golang:1.21 AS builder

WORKDIR /building
COPY . .

ARG APP
RUN make ${APP}

FROM alpine:3.18 AS runtime

ARG APP
RUN addgroup -g 1000 -S ${APP} && adduser -u 1000 -S ${APP} -G ${APP} --home /app \
 && echo -e "#!/bin/sh\nexec /usr/local/bin/${APP} \$@" > /app/entrypoint.sh \
 && chmod +x /app/entrypoint.sh

FROM alpine:3.18

ARG APP
ARG TITLE
LABEL org.opencontainers.image.authors="fatedier <fatedier@gmail.com>"
LABEL org.opencontainers.image.base.name="docker.io/library/alpine:3.18"
LABEL org.opencontainers.image.description="A fast reverse proxy to help you expose a local server behind a NAT or firewall to the internet."
LABEL org.opencontainers.image.licenses="Apache-2.0"
LABEL org.opencontainers.image.source="https://github.com/fatedier/frp"
LABEL org.opencontainers.image.title="${TITLE}"

WORKDIR /
COPY --from=runtime /etc/passwd /etc/group /etc/
COPY --from=runtime --chown=1000:1000 /app/ /app/
COPY --from=builder --chown=1000:1000 /building/bin/${APP} /usr/local/bin/

USER ${APP}

ENTRYPOINT ["/app/entrypoint.sh"]
