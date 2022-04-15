export PATH := $(GOPATH)/bin:$(PATH)
export GO111MODULE=on
LDFLAGS := -s -w

all: fmt build

build: frps frpc

# compile assets into binary file
file:
	rm -rf ./assets/frps/static/*
	rm -rf ./assets/frpc/static/*
	cp -rf ./web/frps/dist/* ./assets/frps/static
	cp -rf ./web/frpc/dist/* ./assets/frpc/static

fmt:
	go fmt ./...

vet:
	go vet ./...

frps:
	env CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o bin/frps ./cmd/frps

frpc:
	env CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o bin/frpc ./cmd/frpc

test: gotest

gotest:
	go test -v --cover ./assets/...
	go test -v --cover ./cmd/...
	go test -v --cover ./client/...
	go test -v --cover ./server/...
	go test -v --cover ./pkg/...

e2e:
	./hack/run-e2e.sh

e2e-trace:
	DEBUG=true LOG_LEVEL=trace ./hack/run-e2e.sh

alltest: vet gotest e2e
	
clean:
	rm -f ./bin/frpc
	rm -f ./bin/frps
