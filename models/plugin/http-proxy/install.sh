#!/bin/bash

export GOPATH=$GOPATH:$PWD

cd $PWD/src/main
go build -o http-proxy

echo "successfully build,binary executable file in src/main"
