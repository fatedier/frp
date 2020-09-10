export PATH := $(GOPATH)/bin:$(PATH)
export GO111MODULE=on

all: fmt build

build: frps frpc

# compile assets into binary file
file:
	rm -rf ./assets/frps/static/*
	rm -rf ./assets/frpc/static/*
	cp -rf ./web/frps/dist/* ./assets/frps/static
	cp -rf ./web/frpc/dist/* ./assets/frpc/static
	rm -rf ./assets/frps/statik
	rm -rf ./assets/frpc/statik
	go generate ./assets/...

fmt:
	go fmt ./...

frps:
	go build -o bin/frps ./cmd/frps

frpc:
	go build -o bin/frpc ./cmd/frpc

test: gotest

gotest:
	go test -v --cover ./assets/...
	go test -v --cover ./client/...
	go test -v --cover ./cmd/...
	go test -v --cover ./models/...
	go test -v --cover ./server/...
	go test -v --cover ./utils/...

ci:
	go test -count=1 -p=1 -v ./tests/...

alltest: gotest ci
	
clean:
	rm -f ./bin/frpc
	rm -f ./bin/frps
