export PATH := $(PATH):`go env GOPATH`/bin
export GO111MODULE=on
LDFLAGS := -s -w
NOWEB_TAG = $(shell [ ! -d web/frps/dist ] || [ ! -d web/frpc/dist ] && echo ',noweb')
FRP_COMPAT_BASELINE_COUNT ?= 8
FRP_COMPAT_FLOOR_VERSION ?= 0.61.0

# Size-optimized build flags (noweb + stripped + no buildid)
SLIM_LDFLAGS := -s -w -buildid=
SLIM_TAGS_FRPC := frpc,noweb
SLIM_TAGS_FRPS := frps,noweb
OUT_BIN_FRPC ?= bin/frpc_slim
OUT_BIN_FRPS ?= bin/frps_slim

# UPX_COMPRESS=1 (default) compresses the slim binaries with UPX if installed.
# Set UPX_COMPRESS=0 to skip compression.
UPX_COMPRESS ?= 1

.PHONY: web frps-web frpc-web frps frpc e2e-compatibility-smoke e2e-compatibility e2e-compatibility-floor frpc-mt7621 frps-mt7621 frpc-slim frps-slim

all: env fmt web build

build: frps frpc

env:
	@go version

web: frps-web frpc-web

frps-web:
	$(MAKE) -C web/frps build

frpc-web:
	$(MAKE) -C web/frpc build

fmt:
	go fmt ./...

fmt-more:
	gofumpt -l -w .

gci:
	gci write -s standard -s default -s "prefix(github.com/fatedier/frp/)" ./

vet:
	go vet -tags "$(NOWEB_TAG)" ./...

frps:
	env CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -tags "frps$(NOWEB_TAG)" -o bin/frps ./cmd/frps

frpc:
	env CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -tags "frpc$(NOWEB_TAG)" -o bin/frpc ./cmd/frpc

# Slim builds (stripped, without web admin, optionally UPX-compressed)
frpc-slim:
	env CGO_ENABLED=0 go build -trimpath -buildvcs=false -ldflags "$(SLIM_LDFLAGS)" \
		-tags "$(SLIM_TAGS_FRPC)" -o $(OUT_BIN_FRPC) ./cmd/frpc
	@if [ "$(UPX_COMPRESS)" = "1" ]; then \
		if command -v upx >/dev/null 2>&1; then \
			echo "Compressing $(OUT_BIN_FRPC) with UPX..."; \
			upx --ultra-brute --lzma $(OUT_BIN_FRPC); \
		else \
			echo "WARN: upx not found in PATH, skipping compression."; \
		fi; \
	fi
	@ls -lh $(OUT_BIN_FRPC)

frps-slim:
	env CGO_ENABLED=0 go build -trimpath -buildvcs=false -ldflags "$(SLIM_LDFLAGS)" \
		-tags "$(SLIM_TAGS_FRPS)" -o $(OUT_BIN_FRPS) ./cmd/frps
	@if [ "$(UPX_COMPRESS)" = "1" ]; then \
		if command -v upx >/dev/null 2>&1; then \
			echo "Compressing $(OUT_BIN_FRPS) with UPX..."; \
			upx --ultra-brute --lzma $(OUT_BIN_FRPS); \
		else \
			echo "WARN: upx not found in PATH, skipping compression."; \
		fi; \
	fi
	@ls -lh $(OUT_BIN_FRPS)

# MT7621 router build (MIPS little-endian, softfloat) — minimal size.
# OpenWrt on MT7621 uses softfloat ABI, so this is the safe default.
# Set UPX_COMPRESS=0 to skip UPX compression.
frpc-mt7621:
	env GOOS=linux GOARCH=mipsle GOMIPS=softfloat $(MAKE) frpc-slim OUT_BIN_FRPC=bin/frpc_mt7621

frps-mt7621:
	env GOOS=linux GOARCH=mipsle GOMIPS=softfloat $(MAKE) frps-slim OUT_BIN_FRPS=bin/frps_mt7621

test: gotest

gotest:
	go test -tags "$(NOWEB_TAG)" -v --cover ./assets/...
	go test -tags "$(NOWEB_TAG)" -v --cover ./cmd/...
	go test -tags "$(NOWEB_TAG)" -v --cover ./client/...
	go test -tags "$(NOWEB_TAG)" -v --cover ./server/...
	go test -tags "$(NOWEB_TAG)" -v --cover ./pkg/...

e2e:
	./hack/run-e2e.sh

e2e-trace:
	DEBUG=true LOG_LEVEL=trace ./hack/run-e2e.sh

e2e-compatibility-smoke: build
	FRP_COMPAT_BASELINE_COUNT=1 ./hack/run-e2e-compatibility.sh

e2e-compatibility: build
	FRP_COMPAT_BASELINE_COUNT="$(FRP_COMPAT_BASELINE_COUNT)" ./hack/run-e2e-compatibility.sh

e2e-compatibility-floor: build
	FRP_COMPAT_BASELINE_VERSIONS="$(FRP_COMPAT_FLOOR_VERSION)" ./hack/run-e2e-compatibility.sh

e2e-compatibility-last-frpc:
	if [ ! -d "./lastversion" ]; then \
		TARGET_DIRNAME=lastversion ./hack/download.sh; \
	fi
	FRPC_PATH="`pwd`/lastversion/frpc" ./hack/run-e2e.sh
	rm -r ./lastversion

e2e-compatibility-last-frps:
	if [ ! -d "./lastversion" ]; then \
		TARGET_DIRNAME=lastversion ./hack/download.sh; \
	fi
	FRPS_PATH="`pwd`/lastversion/frps" ./hack/run-e2e.sh
	rm -r ./lastversion

alltest: vet gotest e2e
	
clean:
	rm -f ./bin/frpc
	rm -f ./bin/frps
	rm -rf ./lastversion
	rm -rf ./.cache
	rm -rf ./.compat
