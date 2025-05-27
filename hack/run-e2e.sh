#!/bin/sh

SCRIPT=$(readlink -f "$0")
ROOT=$(unset CDPATH && cd "$(dirname "$SCRIPT")/.." && pwd)

# Check if ginkgo is available
if ! command -v ginkgo >/dev/null 2>&1; then
    echo "ginkgo not found, try to install..."
    go install github.com/onsi/ginkgo/v2/ginkgo@v2.23.4
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
concurrency="16"
if [ "${CONCURRENCY}" ]; then
    concurrency="${CONCURRENCY}"
fi

ginkgo -nodes=${concurrency} --poll-progress-after=60s ${ROOT}/test/e2e -- -frpc-path=${frpcPath} -frps-path=${frpsPath} -log-level=${logLevel} -debug=${debug}
