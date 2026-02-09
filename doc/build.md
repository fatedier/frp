# frp 编译指南

## 环境要求

- Go 1.24+
- Node.js (如需编译 Web UI)

## 快速开始

### 1. 设置国内代理（推荐）

```bash
go env -w GOPROXY=https://goproxy.cn,direct
```

### 2. 下载依赖

```bash
go mod tidy
```

### 3. 创建 Web UI 占位文件（不编译 Web UI 时）

```bash
mkdir -p web/frps/dist web/frpc/dist
echo "placeholder" > web/frps/dist/index.html
echo "placeholder" > web/frpc/dist/index.html
```

### 4. 编译

```bash
# 编译 frps
go build -o frps ./cmd/frps

# 编译 frpc
go build -o frpc ./cmd/frpc

# 一次性编译两个
go build -o frps ./cmd/frps && go build -o frpc ./cmd/frpc
```

## 完整编译（包含 Web UI）

```bash
# 编译 frps Web UI
cd web/frps && npm install && npm run build && cd ../..

# 编译 frpc Web UI
cd web/frpc && npm install && npm run build && cd ../..

# 编译 Go 二进制
go build -o frps ./cmd/frps
go build -o frpc ./cmd/frpc
```

## 交叉编译

```bash
# Linux amd64
GOOS=linux GOARCH=amd64 go build -o frps_linux_amd64 ./cmd/frps
GOOS=linux GOARCH=amd64 go build -o frpc_linux_amd64 ./cmd/frpc

# Linux arm64
GOOS=linux GOARCH=arm64 go build -o frps_linux_arm64 ./cmd/frps
GOOS=linux GOARCH=arm64 go build -o frpc_linux_arm64 ./cmd/frpc

# Windows amd64
GOOS=windows GOARCH=amd64 go build -o frps_windows_amd64.exe ./cmd/frps
GOOS=windows GOARCH=amd64 go build -o frpc_windows_amd64.exe ./cmd/frpc

# macOS arm64 (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -o frps_darwin_arm64 ./cmd/frps
GOOS=darwin GOARCH=arm64 go build -o frpc_darwin_arm64 ./cmd/frpc
```

## 一键脚本

```bash
#!/bin/bash

# 设置代理
go env -w GOPROXY=https://goproxy.cn,direct

# 下载依赖
go mod tidy

# 创建占位文件
mkdir -p web/frps/dist web/frpc/dist
echo "placeholder" > web/frps/dist/index.html
echo "placeholder" > web/frpc/dist/index.html

# 编译
go build -o frps ./cmd/frps
go build -o frpc ./cmd/frpc

echo "编译完成！"
ls -la frps frpc
```

## 常见问题

### 1. 报错 `pattern dist: no matching files found`

创建占位文件：
```bash
mkdir -p web/frps/dist web/frpc/dist
echo "placeholder" > web/frps/dist/index.html
echo "placeholder" > web/frpc/dist/index.html
```

### 2. 依赖下载慢

设置国内代理：
```bash
go env -w GOPROXY=https://goproxy.cn,direct
```

### 3. 缺少 go.sum 条目

运行：
```bash
go mod tidy
```


etcd 中 token 配置的 JSON 格式：


{
  "token": "your_token_string",
  "region": "chengdu",
  "allowPorts": [
    {"single": 8080},
    {"start": 9000, "end": 9100}
  ],
  "bandwidthLimit": "10MB",
  "maxPortsPerClient": 5,
  "enabled": true,
  "description": "可选的描述信息"
}
字段说明：

字段	类型	必填	说明
token	string	✅	认证 token，frpc 使用此值
region	string	✅	区域，必须与 frps 的 etcd.region 匹配
allowPorts	array	❌	允许的端口，空则不限制
bandwidthLimit	string	❌	带宽限制，如 10MB, 1024KB
maxPortsPerClient	int	❌	最大端口数，0 表示不限制
enabled	bool	✅	是否启用，false 则拒绝连接
description	string	❌	描述信息
示例命令：


# 添加/更新 token
etcdctl put /frp/tokens/my_token '{
  "token": "my_token",
  "region": "chengdu",
  "allowPorts": [{"single": 8080}, {"start": 9000, "end": 9100}],
  "bandwidthLimit": "10MB",
  "maxPortsPerClient": 5,
  "enabled": true,
  "description": "用户A"
}'

# 禁用 token（会断开已有连接）
etcdctl put /frp/tokens/my_token '{
  "token": "my_token",
  "region": "chengdu",
  "enabled": false
}'

# 删除 token（会断开已有连接）
etcdctl del /frp/tokens/my_token

# 查看 token
etcdctl get /frp/tokens/my_token

# 查看所有 token
etcdctl get /frp/tokens/ --prefix
端口格式：

单端口: {"single": 8080}
端口范围: {"start": 9000, "end": 9100}
