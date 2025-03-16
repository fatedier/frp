### Features

- Go version update to 1.24.1
- Security updates

### Fixes

- GoReleaser workflow update to specify version range instead of latest
- GoLangCI workflow update to output format
- cmd/frps/root.go & cmd/frpc/root.go updates use .toml instead of .ini
- Removal of SetStreamMode(true) from pkg/util/net/kcp.go as it is default behavior and the function is being deprecated

### Dependency Updates

- `vite` from 5.0.12 to 5.4.12
- `braces` from 3.0.2 to 3.0.3
- `esbuild` from 0.19.12 to 0.21.5
- `nanoid` from 3.3.7 to 3.3.8
- `rollup` from 4.9.6 to 4.34.9
- github.com/cespare/xxhash/v2 from 2.2.0 to 2.3.0
- github.com/coreos/go-oidc/v3 from 3.10.0 to 3.12.0
- github.com/cpuguy83/go-md2man/v2 from 2.0.3 to 2.0.6
- github.com/davecgh/go-spew from 1.1.1 to 1.1.2-0.20180830191138-d8f796af33cc
- github.com/go-jose/go-jose/v4 from 4.0.1 to 4.0.5
- github.com/gorilla/websocket from 1.5.0 to 1.5.3
- github.com/onsi/ginkgo/v2 from 2.22.0 to 2.23.0
- github.com/onsi/gomega from 1.34.2 to 1.36.2
- github.com/pelletier/go-toml/v2 from 2.2.0 to 2.2.3
- github.com/pires/go-proxyproto from 0.7.0 to 0.8.0
- github.com/prometheus/client_golang from 1.19.1 to 1.21.1
- github.com/prometheus/client_model from 0.5.0 to 0.6.1
- github.com/prometheus/common from 0.48.0 to 0.62.0
- github.com/prometheus/procfs from 0.12.0 to 0.15.1
- github.com/pmezard/go-difflib from 1.0.0 to 1.0.1-0.20181226105442-5d4384ee4fb2
- github.com/quic-go/quic-go from 0.48.2 to 0.50.0
- github.com/rodaine/table from 1.2.0 to 1.3.0
- github.com/samber/lo from 1.47.0 to 1.49.1
- github.com/spf13/pflag from 1.0.5 to 1.0.6
- github.com/spf13/cobra from 1.8.0 to 1.9.1
- github.com/stretchr/testify from 1.9.0 to 1.10.0
- github.com/tidwall/gjson from 1.17.1 to 1.18.0
- github.com/xtaci/kcp-go/v5 from 5.6.13 to 5.6.18
- golang.org/x/crypto from 0.30.0 to 0.36.0
- golang.org/x/mod from 0.22.0 to 0.23.0
- golang.org/x/net from 0.32.0 to 0.37.0
- golang.org/x/oauth2 from 0.16.0 to 0.28.0
- golang.org/x/sync from 0.10.0 to 0.12.0
- golang.org/x/sys from 0.29.0 to 0.31.0
- golang.org/x/text from 0.21.0 to 0.23.0
- golang.org/x/term from 0.28.0 to 0.30.0
- golang.org/x/time from 0.5.0 to 0.11.0
- golang.org/x/tools from 0.28.0 to 0.30.0
- google.golang.org/protobuf from 1.35.1 to 1.36.1
- k8s.io/client-go from 0.28.8 to 0.32.2
- k8s.io/apimachinery from 0.28.8 to 0.32.2
- k8s.io/utils from 0.0.0-20230406110748-d93618cff8a2 to 0.0.0-20241104100929-3ea5e8cea738
- sigs.k8s.io/json from 0.0.0-20221116044647-bc3834ca7abd to 0.0.0-20241010143419-9aa6b5e7a4b3
- sigs.k8s.io/yaml from 1.3.0 to 1.4.0
