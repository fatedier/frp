#!/bin/sh

OS="$(go env GOOS)"
ARCH="$(go env GOARCH)"

if [ "${TARGET_OS}" ]; then
  OS="${TARGET_OS}"
fi
if [ "${TARGET_ARCH}" ]; then
  ARCH="${TARGET_ARCH}"
fi

# Determine the latest version by version number ignoring alpha, beta, and rc versions.
if [ "${FRP_VERSION}" = "" ] ; then
  FRP_VERSION="$(curl -sL https://github.com/fatedier/frp/releases | \
                  grep -o 'releases/tag/v[0-9]*.[0-9]*.[0-9]*"' | sort -V | \
                  tail -1 | awk -F'/' '{ print $3}')"
  FRP_VERSION="${FRP_VERSION%?}"
  FRP_VERSION="${FRP_VERSION#?}"
fi

if [ "${FRP_VERSION}" = "" ] ; then
  printf "Unable to get latest frp version. Set FRP_VERSION env var and re-run. For example: export FRP_VERSION=1.0.0"
  exit 1;
fi

SUFFIX=".tar.gz"
if [ "${OS}" = "windows" ] ; then
  SUFFIX=".zip"
fi
NAME="frp_${FRP_VERSION}_${OS}_${ARCH}${SUFFIX}"
DIR_NAME="frp_${FRP_VERSION}_${OS}_${ARCH}"
URL="https://github.com/fatedier/frp/releases/download/v${FRP_VERSION}/${NAME}"

download_and_extract() {
  printf "Downloading %s from %s ...\n" "$NAME" "${URL}"
  if ! curl -o /dev/null -sIf "${URL}"; then
    printf "\n%s is not found, please specify a valid FRP_VERSION\n" "${URL}"
    exit 1
  fi
  curl -fsLO "${URL}"
  filename=$NAME

  if [ "${OS}" = "windows" ]; then
    unzip "${filename}"
  else
    tar -xzf "${filename}"
  fi
  rm "${filename}"

  if [ "${TARGET_DIRNAME}" ]; then
    mv "${DIR_NAME}" "${TARGET_DIRNAME}"
    DIR_NAME="${TARGET_DIRNAME}"
  fi
}

download_and_extract

printf ""
printf "\nfrp %s Download Complete!\n" "$FRP_VERSION"
printf "\n"
printf "frp has been successfully downloaded into the %s folder on your system.\n" "$DIR_NAME"
printf "\n"
