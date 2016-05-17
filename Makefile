export PATH := $(GOPATH)/bin:$(PATH)
export OLDGOPATH := $(GOPATH)
export GOPATH := $(shell pwd):$(GOPATH)

all: build

build: godep fmt frps frpc build_test

build_test: echo_server http_server

godep:
	GOPATH=$(OLDGOPATH) go get github.com/tools/godep

fmt:
	go fmt ./src/...
	@go fmt ./test/echo_server.go
	@go fmt ./test/http_server.go
	@go fmt ./test/func_test.go

frps:
	godep go build -o bin/frps ./src/frp/cmd/frps

frpc:
	godep go build -o bin/frpc ./src/frp/cmd/frpc

echo_server:
	godep go build -o test/bin/echo_server ./test/echo_server.go

http_server:
	godep go build -o test/bin/http_server ./test/http_server.go

test: gotest

gotest:
	godep go test -v ./src/...

alltest:
	cd ./test && ./run_test.sh && cd -
	godep go test -v ./src/...
	godep go test -v ./test/func_test.go
	cd ./test && ./clean_test.sh && cd -

clean:
	rm -f ./bin/frpc
	rm -f ./bin/frps
	rm -f ./test/bin/echo_server
	rm -f ./test/bin/http_server
	cd ./test && ./clean_test.sh && cd -
