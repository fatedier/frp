export PATH := $(GOPATH)/bin:$(PATH)
export NEW_GOPATH := $(shell pwd)

all: build

build: godep fmt frps frpc

godep:
	@go get github.com/tools/godep

fmt:
	GOPATH=$(NEW_GOPATH) godep go fmt ./...

frps:
	GOPATH=$(NEW_GOPATH) godep go build -o bin/frps ./src/frp/cmd/frps

frpc:
	GOPATH=$(NEW_GOPATH) godep go build -o bin/frpc ./src/frp/cmd/frpc

test:
	@GOPATH=$(NEW_GOPATH) godep go test -v ./...
