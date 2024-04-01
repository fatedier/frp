export PATH := $(PATH):`go env GOPATH`/bin
export GO111MODULE=on
LDFLAGS := -s -w

.PHONY: all
all: env fmt build

.PHONY: build
build: frps frpc

.PHONY: env
env:
	@go version

# compile assets into binary file
.PHONY: file
file:
	rm -rf ./assets/frps/static/*
	rm -rf ./assets/frpc/static/*
	cp -rf ./web/frps/dist/* ./assets/frps/static
	cp -rf ./web/frpc/dist/* ./assets/frpc/static

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: fmt-more
fmt-more:
	gofumpt -l -w .

.PHONY: gci
gci:
	gci write -s standard -s default -s "prefix(github.com/fatedier/frp/)" ./

.PHONY: vet
vet:
	go vet ./...

.PHONY: frps
frps:
	env CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -tags frps -o bin/frps ./cmd/frps

.PHONY: frpc
frpc:
	env CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -tags frpc -o bin/frpc ./cmd/frpc

.PHONY: test
test: gotest

.PHONY: gotest
gotest:
	go test -v --cover ./assets/...
	go test -v --cover ./cmd/...
	go test -v --cover ./client/...
	go test -v --cover ./server/...
	go test -v --cover ./pkg/...

.PHONY: e2e
e2e:
	./hack/run-e2e.sh

.PHONY: e2e-trace
e2e-trace:
	DEBUG=true LOG_LEVEL=trace ./hack/run-e2e.sh

.PHONY: e2e-compatibility-last-frpc
e2e-compatibility-last-frpc:
	if [ ! -d "./lastversion" ]; then \
		TARGET_DIRNAME=lastversion ./hack/download.sh; \
	fi
	FRPC_PATH="`pwd`/lastversion/frpc" ./hack/run-e2e.sh
	rm -r ./lastversion

.PHONY: e2e-compatibility-last-frps
e2e-compatibility-last-frps:
	if [ ! -d "./lastversion" ]; then \
		TARGET_DIRNAME=lastversion ./hack/download.sh; \
	fi
	FRPS_PATH="`pwd`/lastversion/frps" ./hack/run-e2e.sh
	rm -r ./lastversion

.PHONY: alltest
alltest: vet gotest e2e

.PHONY: clean
clean:
	rm -f ./bin/frpc
	rm -f ./bin/frps
	rm -rf ./lastversion
