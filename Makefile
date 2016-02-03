export PATH := $(GOPATH)/bin:$(PATH)

all: build

build: godep frps frpc

godep:
	@go get github.com/tools/godep
	godep restore

frps:
	godep go build -o bin/frps ./cmd/frps

frpc:
	godep go build -o bin/frpc ./cmd/frpc

test:
	@godep go test ./...
