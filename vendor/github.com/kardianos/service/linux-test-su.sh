#!/usr/bin/env bash
# This script is used to run the tests under linux as root
#
# Usage:
#    linux-test-su.sh goPath goBinPath
#
# goPath is the standard GOPATH
# goBinPath is the location of go
#
# Typical usage:
#    sudo ./linux-test-su.sh $GOPATH `which go`

export GOPATH=$1
export GOROOT=`dirname $(dirname $2)`
$GOROOT/bin/go test -v -tags su ./...
