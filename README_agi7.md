## build nvr frpc
```shell
env GOOS=linux GOARCH=arm GOARM=7 go build -v -o frpc ./cmd/frpc
```

## build frps for linux
```shell
env GOOS=linux GOARCH=amd64 go build -v -o frpc ./cmd/frps
```
