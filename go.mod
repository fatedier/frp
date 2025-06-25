module github.com/fatedier/frp

go 1.23.0

require (
	github.com/armon/go-socks5 v0.0.0-20160902184237-e75332964ef5
	github.com/coreos/go-oidc/v3 v3.14.1
	github.com/fatedier/golib v0.5.1
	github.com/google/uuid v1.6.0
	github.com/gorilla/mux v1.8.1
	github.com/gorilla/websocket v1.5.0
	github.com/hashicorp/yamux v0.1.1
	github.com/onsi/ginkgo/v2 v2.23.4
	github.com/onsi/gomega v1.36.3
	github.com/pelletier/go-toml/v2 v2.2.0
	github.com/pion/stun/v2 v2.0.0
	github.com/pires/go-proxyproto v0.7.0
	github.com/prometheus/client_golang v1.19.1
	github.com/quic-go/quic-go v0.48.2
	github.com/rodaine/table v1.2.0
	github.com/samber/lo v1.47.0
	github.com/songgao/water v0.0.0-20200317203138-2b4b6d7c09d8
	github.com/spf13/cobra v1.8.0
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.10.0
	github.com/tidwall/gjson v1.17.1
	github.com/vishvananda/netlink v1.3.0
	github.com/xtaci/kcp-go/v5 v5.6.13
	golang.org/x/crypto v0.37.0
	golang.org/x/net v0.39.0
	golang.org/x/oauth2 v0.28.0
	golang.org/x/sync v0.13.0
	golang.org/x/time v0.5.0
	golang.zx2c4.com/wireguard v0.0.0-20231211153847-12269c276173
	gopkg.in/ini.v1 v1.67.0
	k8s.io/apimachinery v0.28.8
	k8s.io/client-go v0.28.8
)

require (
	github.com/Azure/go-ntlmssp v0.0.0-20221128193559-754e69321358 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/go-jose/go-jose/v4 v4.0.5 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-task/slim-sprig/v3 v3.0.0 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/google/pprof v0.0.0-20250403155104-27863c87afa6 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/klauspost/cpuid/v2 v2.2.6 // indirect
	github.com/klauspost/reedsolomon v1.12.0 // indirect
	github.com/pion/dtls/v2 v2.2.7 // indirect
	github.com/pion/logging v0.2.2 // indirect
	github.com/pion/transport/v2 v2.2.1 // indirect
	github.com/pion/transport/v3 v3.0.1 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/client_model v0.5.0 // indirect
	github.com/prometheus/common v0.48.0 // indirect
	github.com/prometheus/procfs v0.12.0 // indirect
	github.com/templexxx/cpu v0.1.1 // indirect
	github.com/templexxx/xorsimd v0.4.3 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.0 // indirect
	github.com/tjfoc/gmsm v1.4.1 // indirect
	github.com/vishvananda/netns v0.0.4 // indirect
	go.uber.org/automaxprocs v1.6.0 // indirect
	go.uber.org/mock v0.5.0 // indirect
	golang.org/x/exp v0.0.0-20241204233417-43b7b7cde48d // indirect
	golang.org/x/mod v0.24.0 // indirect
	golang.org/x/sys v0.32.0 // indirect
	golang.org/x/text v0.24.0 // indirect
	golang.org/x/tools v0.31.0 // indirect
	golang.zx2c4.com/wintun v0.0.0-20230126152724-0fa3db229ce2 // indirect
	google.golang.org/protobuf v1.36.5 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	k8s.io/utils v0.0.0-20230406110748-d93618cff8a2 // indirect
	sigs.k8s.io/json v0.0.0-20221116044647-bc3834ca7abd // indirect
	sigs.k8s.io/yaml v1.3.0 // indirect
)

// TODO(fatedier): Temporary use the modified version, update to the official version after merging into the official repository.
replace github.com/hashicorp/yamux => github.com/fatedier/yamux v0.0.0-20230628132301-7aca4898904d
