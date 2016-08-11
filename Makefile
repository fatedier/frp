export PATH := $(GOPATH)/bin:$(PATH)
export GO15VENDOREXPERIMENT := 1

all: fmt dep build

build: frps frpc build_test

build_test: echo_server http_server

dep: statik

statik:
	go get -d github.com/rakyll/statik
	@go install github.com/rakyll/statik
	@rm -rf ./src/assets/statik
	go generate ./src/...

fmt:
	go fmt ./src/...
	@go fmt ./test/echo_server.go
	@go fmt ./test/http_server.go
	@go fmt ./test/func_test.go

frps:
	go build -o bin/frps ./src/cmd/frps
	cp -rf ./src/assets/static ./bin

frpc:
	go build -o bin/frpc ./src/cmd/frpc

echo_server:
	go build -o test/bin/echo_server ./test/echo_server.go

http_server:
	go build -o test/bin/http_server ./test/http_server.go

test: gotest

gotest:
	go test -v ./src/...

alltest:
	cd ./test && ./run_test.sh && cd -
	go test -v ./src/...
	go test -v ./test/func_test.go
	cd ./test && ./clean_test.sh && cd -

clean:
	rm -f ./bin/frpc
	rm -f ./bin/frps
	rm -f ./test/bin/echo_server
	rm -f ./test/bin/http_server
	cd ./test && ./clean_test.sh && cd -

save:
	godep save ./src/...
