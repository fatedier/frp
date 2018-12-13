export PATH := $(GOPATH)/bin:$(PATH)

all: fmt build

build: frps frpc

# compile assets into binary file
file:
	rm -rf ./assets/static/*
	cp -rf ./web/frps/dist/* ./assets/static
	go get -d github.com/rakyll/statik
	go install github.com/rakyll/statik
	rm -rf ./assets/statik
	go generate ./assets/...

fmt:
	go fmt ./...
	
frps:
	go build -o bin/frps ./cmd/frps
	@cp -rf ./assets/static ./bin

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
