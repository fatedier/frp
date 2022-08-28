#!/usr/bin/env bash

ROOT=$(unset CDPATH && cd $(dirname "${BASH_SOURCE[0]}")/.. && pwd)

which ginkgo &> /dev/null
if [ $? -ne 0 ]; then
    echo "ginkgo not found, try to install..."
    go install github.com/onsi/ginkgo/ginkgo@v1.16.5
fi

debug=false
if [ x${DEBUG} == x"true" ]; then
    debug=true
fi
logLevel=debug
if [ x${LOG_LEVEL} != x"" ]; then
    logLevel=${LOG_LEVEL}
fi

ginkgo -nodes=8 -slowSpecThreshold=20 ${ROOT}/test/e2e -- -frpc-path=${ROOT}/bin/frpc -frps-path=${ROOT}/bin/frps -log-level=${logLevel} -debug=${debug}
