#!/usr/bin/env bash

ROOT=$(unset CDPATH && cd $(dirname "${BASH_SOURCE[0]}")/.. && pwd)

which ginkgo &> /dev/null
if [ $? -ne 0 ]; then
    echo "ginkgo not found, try to install..."
    go install github.com/onsi/ginkgo/v2/ginkgo@v2.8.3
fi

debug=false
if [ x${DEBUG} == x"true" ]; then
    debug=true
fi
logLevel=debug
if [ x${LOG_LEVEL} != x"" ]; then
    logLevel=${LOG_LEVEL}
fi

ginkgo -nodes=8 --poll-progress-after=20s ${ROOT}/test/e2e -- -frpc-path=${ROOT}/bin/frpc -frps-path=${ROOT}/bin/frps -log-level=${logLevel} -debug=${debug}
