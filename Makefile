export PATH := $(GOPATH)/bin:$(PATH)

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

frps: frps-arm frps-arm64 frps-amd64

frps-arm:
	GOARCH=arm go build -o bin/arm/frps ./cmd/frps

frps-arm64:
	GOARCH=arm64 go build -o bin/arm64/frps ./cmd/frps

frps-amd64:
	GOARCH=amd64 go build -o bin/amd64/frps ./cmd/frps

frpc: frpc-arm frpc-arm64 frpc-amd64

frpc-arm:
	GOOS=linux GOARCH=arm go build -o bin/arm/frpc ./cmd/frpc

frpc-arm64:
	GOOS=linux GOARCH=arm64 go build -o bin/arm64/frpc ./cmd/frpc

frpc-amd64:
	GOOS=linux GOARCH=amd64 go build -o bin/amd64/frpc ./cmd/frpc

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
