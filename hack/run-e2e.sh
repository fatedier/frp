#!/bin/sh

SCRIPT=$(readlink -f "$0")
ROOT=$(unset CDPATH && cd "$(dirname "$SCRIPT")/.." && pwd)

ginkgo_command=$(which ginkgo 2>/dev/null)
if [ -z "$ginkgo_command" ]; then
    echo "ginkgo not found, try to install..."
    go install github.com/onsi/ginkgo/v2/ginkgo@v2.8.3
fi

debug=false
if [ "x${DEBUG}" = "xtrue" ]; then
    debug=true
fi
logLevel=debug
if [ "${LOG_LEVEL}" ]; then
    logLevel="${LOG_LEVEL}"
fi

frpcPath=${ROOT}/bin/frpc
if [ "${FRPC_PATH}" ]; then
    frpcPath="${FRPC_PATH}"
fi
frpsPath=${ROOT}/bin/frps
if [ "${FRPS_PATH}" ]; then
    frpsPath="${FRPS_PATH}"
fi

ginkgo -nodes=8 --poll-progress-after=60s ${ROOT}/test/e2e -- -frpc-path=${frpcPath} -frps-path=${frpsPath} -log-level=${logLevel} -debug=${debug}
