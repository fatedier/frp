ARG BUILD_IMAGE="golang:1.16"

FROM ${BUILD_IMAGE} as builder

COPY . /app/src

WORKDIR /app/src

ENV \
    GO111MODULE="auto" \
    GOPROXY=https://goproxy.cn,direct \
    GOCACHE=/var/cache/go-build

RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories \
    && apk add --no-cache binutils make libcap

RUN \
    make build \
    && mv /app/src/bin/frpc /app/frpc \
    && chmod +x /app/frpc \
    && mv /app/src/bin/frps /app/frps \
    && chmod +x /app/frps

RUN \
    setcap cap_net_bind_service=+eip /app/frps \
    && getcap /app/frps \
    && setcap cap_net_bind_service=+eip /app/frpc \
    && getcap /app/frpc

RUN \
    strip /app/frpc \
    && strip /app/frps

FROM scratch

ARG RUN_AS_USER=0
ARG RUN_AS_GROUP=0
ARG GIT_COMMIT=""
ARG GIT_BRANCH=""
ARG BUILD_DATE=""

LABEL \
    GIT_COMMIT=${GIT_COMMIT} \
    GIT_BRANCH=${GIT_BRANCH} \
    BUILD_DATE=${BUILD_DATE} \
    RUN_AS_USER=${RUN_AS_USER} \
    RUN_AS_GROUP=${RUN_AS_GROUP}

COPY --from=builder /app/frpc /usr/bin/frpc
COPY --from=builder /app/frps /usr/bin/frps

USER ${RUN_AS_USER}:${RUN_AS_GROUP}

CMD [ "/app/frpc" ]
