#!/bin/sh

set -eu

SCRIPT=$(readlink -f "$0")
ROOT=$(unset CDPATH && cd "$(dirname "$SCRIPT")/.." && pwd)

if ! command -v ginkgo >/dev/null 2>&1; then
    echo "ginkgo not found, try to install..."
    go install github.com/onsi/ginkgo/v2/ginkgo@v2.23.4
fi

debug=false
if [ "x${DEBUG:-}" = "xtrue" ]; then
    debug=true
fi
logLevel=debug
if [ "${LOG_LEVEL:-}" ]; then
    logLevel="${LOG_LEVEL}"
fi

currentFrpsPath=${CURRENT_FRPS_PATH:-${ROOT}/bin/frps}
currentFrpcPath=${CURRENT_FRPC_PATH:-${ROOT}/bin/frpc}
baselineCount=${FRP_COMPAT_BASELINE_COUNT:-8}
targetOS=${TARGET_OS:-$(go env GOOS)}
targetArch=${TARGET_ARCH:-$(go env GOARCH)}
targetPlatform="${targetOS}_${targetArch}"
cacheRoot=${FRP_COMPAT_CACHE_DIR:-${ROOT}/.cache/e2e-compat}

check_file() {
    if [ ! -f "$2" ]; then
        echo "$1 not found: $2"
        exit 1
    fi
}

check_file "current frps" "${currentFrpsPath}"
check_file "current frpc" "${currentFrpcPath}"

run_current_current=true

run_compatibility() {
    baselineVersion=$1
    baselineFrpsPath=$2
    baselineFrpcPath=$3

    check_file "baseline frps" "${baselineFrpsPath}"
    check_file "baseline frpc" "${baselineFrpcPath}"

    echo "Running compatibility e2e with baseline ${baselineVersion}"
    ginkgo -nodes=1 --poll-progress-after=60s "${ROOT}/test/e2e/compatibility" -- \
        -current-frps-path="${currentFrpsPath}" \
        -current-frpc-path="${currentFrpcPath}" \
        -baseline-frps-path="${baselineFrpsPath}" \
        -baseline-frpc-path="${baselineFrpcPath}" \
        -baseline-version="${baselineVersion}" \
        -run-current-current="${run_current_current}" \
        -log-level="${logLevel}" \
        -debug="${debug}"
    run_current_current=false
}

github_api_curl() {
    if [ "${GITHUB_TOKEN:-}" ]; then
        curl -fsSL \
            -H "Accept: application/vnd.github+json" \
            -H "Authorization: Bearer ${GITHUB_TOKEN}" \
            "$1"
    else
        curl -fsSL "$1"
    fi
}

resolve_versions() {
    if [ "${FRP_COMPAT_BASELINE_VERSIONS:-}" ]; then
        printf "%s\n" "${FRP_COMPAT_BASELINE_VERSIONS}"
        return
    fi

    case "${baselineCount}" in
        '' | *[!0-9]*)
            echo "FRP_COMPAT_BASELINE_COUNT must be a positive integer: ${baselineCount}" >&2
            exit 1
            ;;
    esac
    if [ "${baselineCount}" -eq 0 ]; then
        echo "FRP_COMPAT_BASELINE_COUNT must be greater than 0" >&2
        exit 1
    fi
    if [ "${baselineCount}" -gt 100 ]; then
        echo "FRP_COMPAT_BASELINE_COUNT must be less than or equal to 100" >&2
        exit 1
    fi

    releaseURL="https://api.github.com/repos/fatedier/frp/releases?per_page=100"
    resolvedVersions=""
    if releases=$(github_api_curl "${releaseURL}" 2>/dev/null); then
        resolvedVersions=$(printf "%s\n" "${releases}" |
            sed -n 's/.*"tag_name":[[:space:]]*"v\([0-9][0-9]*\.[0-9][0-9]*\.[0-9][0-9]*\)".*/\1/p' |
            awk '!seen[$0]++' |
            head -n "${baselineCount}" |
            tr '\n' ' ' |
            sed 's/[[:space:]]*$//')
    else
        echo "Failed to fetch release metadata from GitHub API, falling back to GitHub releases page." >&2
    fi

    if [ -z "${resolvedVersions}" ]; then
        releasesPageURL="https://github.com/fatedier/frp/releases"
        if ! releases=$(curl -fsSL "${releasesPageURL}"); then
            echo "Failed to fetch release metadata from GitHub: ${releasesPageURL}" >&2
            echo "Set FRP_COMPAT_BASELINE_VERSIONS to run with explicit baseline versions." >&2
            exit 1
        fi
        resolvedVersions=$(printf "%s\n" "${releases}" |
            grep -o 'releases/tag/v[0-9][0-9]*\.[0-9][0-9]*\.[0-9][0-9]*"' |
            sed 's#.*/v##; s/"$//' |
            awk '!seen[$0]++' |
            head -n "${baselineCount}" |
            tr '\n' ' ' |
            sed 's/[[:space:]]*$//')
    fi

    set -- ${resolvedVersions}
    if [ "$#" -lt "${baselineCount}" ]; then
        echo "Only resolved $# stable release versions from GitHub, expected ${baselineCount}." >&2
        echo "Set FRP_COMPAT_BASELINE_VERSIONS to run with explicit baseline versions." >&2
        exit 1
    fi

    printf "%s\n" "${resolvedVersions}"
}

if [ "${BASELINE_FRPS_PATH:-}" ] || [ "${BASELINE_FRPC_PATH:-}" ]; then
    if [ -z "${BASELINE_FRPS_PATH:-}" ] || [ -z "${BASELINE_FRPC_PATH:-}" ]; then
        echo "BASELINE_FRPS_PATH and BASELINE_FRPC_PATH must be set together"
        exit 1
    fi
    run_compatibility "${FRP_COMPAT_BASELINE_VERSION:-custom}" "${BASELINE_FRPS_PATH}" "${BASELINE_FRPC_PATH}"
    exit 0
fi

versions=$(resolve_versions)
echo "Compatibility baseline versions: ${versions}"

mkdir -p "${cacheRoot}"
for version in ${versions}; do
    baselineDir="${cacheRoot}/${version}/${targetPlatform}"
    if [ ! -f "${baselineDir}/frps" ] || [ ! -f "${baselineDir}/frpc" ]; then
        tmpDir="${cacheRoot}/.download-${version}-${targetPlatform}"
        rm -rf "${tmpDir}"
        (
            cd "${cacheRoot}"
            FRP_VERSION="${version}" TARGET_DIRNAME="$(basename "${tmpDir}")" "${ROOT}/hack/download.sh"
        )
        mkdir -p "$(dirname "${baselineDir}")"
        rm -rf "${baselineDir}"
        mv "${tmpDir}" "${baselineDir}"
    fi

    run_compatibility "${version}" "${baselineDir}/frps" "${baselineDir}/frpc"
done
