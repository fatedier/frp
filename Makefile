export PATH := $(GOPATH)/bin:$(PATH)
export OLDGOPATH := $(GOPATH)
export GOPATH := $(shell pwd):$(GOPATH)

all: build

build: godep fmt frps frpc

godep:
	GOPATH=$(OLDGOPATH) go get github.com/tools/godep

fmt:
	godep go fmt ./...

frps:
	godep go build -o bin/frps ./src/frp/cmd/frps

frpc:
	godep go build -o bin/frpc ./src/frp/cmd/frpc

test:
	godep go test -v ./...
