# etcd 多租户 Token 管理

frp 支持通过 etcd 实现多租户 token 管理，允许为每个 token 配置独立的：
- 允许使用的端口范围
- 带宽限速
- 最大端口数量
- 流量上报

## 架构

```
┌─────────────┐         ┌─────────────┐
│   frpc 1    │────────▶│             │
│  token: A   │         │             │
└─────────────┘         │             │
                        │    frps     │◀────▶ etcd
┌─────────────┐         │  region:    │      (token configs)
│   frpc 2    │────────▶│  chengdu    │
│  token: B   │         │             │         │
└─────────────┘         └─────────────┘         │
                              │                 │
                              ▼                 │
                        ┌─────────────┐         │
                        │ Traffic API │◀────────┘
                        │   Server    │  (流量上报)
                        └─────────────┘
```

## frps 配置

```toml
# frps.toml

bindPort = 7000

[etcd]
# etcd 服务器地址列表
endpoints = ["127.0.0.1:2379", "127.0.0.1:2380"]
# 当前 frps 所在区域，只有 region 匹配的 token 才能连接
region = "chengdu"
# token 存储的 key 前缀，默认 "/frp/tokens/"
prefix = "/frp/tokens/"
# 连接超时时间（秒），默认 5
dialTimeout = 5
# 流量上报地址（可选），POST JSON 请求
trafficReportUrl = "http://your-api-server/traffic/report"
# etcd 认证（可选）
username = ""
password = ""

# TLS 配置（可选）
[etcd.tls]
certFile = ""
keyFile = ""
```

## etcd Token 配置

每个 token 存储在 etcd 中，key 格式为 `{prefix}{token}`。

例如：`/frp/tokens/abc123xyz`

### Token 配置结构

```json
{
  "token": "abc123xyz",
  "region": "chengdu",
  "allowPorts": [
    {"single": 8080},
    {"start": 9000, "end": 9100}
  ],
  "bandwidthLimit": "10MB",
  "maxPortsPerClient": 5,
  "enabled": true,
  "description": "用户A的成都节点token",
  "trafficReportIntervalMB": 50
}
```

### 字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| `token` | string | 认证 token |
| `region` | string | 允许连接的区域 |
| `allowPorts` | array | 允许使用的端口范围 |
| `bandwidthLimit` | string | 带宽限制，如 "10MB", "1024KB" |
| `maxPortsPerClient` | int | 最大端口数量，0 表示不限制 |
| `enabled` | bool | 是否启用此 token |
| `trafficReportIntervalMB` | int | 流量上报间隔(MB)，如 50 表示每 50MB 上报一次，0 表示不上报 |
| `description` | string | 可选描述信息 |

## frpc 配置

frpc 配置保持不变，继续使用标准 token 认证：

```toml
# frpc.toml

serverAddr = "frps.example.com"
serverPort = 7000

auth.method = "token"
auth.token = "abc123xyz"

[[proxies]]
name = "web"
type = "tcp"
localIP = "127.0.0.1"
localPort = 80
remotePort = 8080
```

## 使用示例

### 1. 启动 etcd

```bash
etcd --listen-client-urls http://0.0.0.0:2379 \
     --advertise-client-urls http://127.0.0.1:2379
```

### 2. 添加 Token 配置

```bash
# 为成都区域添加一个 token
etcdctl put /frp/tokens/user1_chengdu '{
  "token": "user1_chengdu",
  "region": "chengdu",
  "allowPorts": [{"single": 8080}, {"start": 9000, "end": 9100}],
  "bandwidthLimit": "10MB",
  "maxPortsPerClient": 5,
  "enabled": true,
  "description": "用户1的成都节点"
}'

# 为上海区域添加另一个 token
etcdctl put /frp/tokens/user1_shanghai '{
  "token": "user1_shanghai",
  "region": "shanghai",
  "allowPorts": [{"start": 10000, "end": 10100}],
  "bandwidthLimit": "5MB",
  "maxPortsPerClient": 3,
  "enabled": true,
  "description": "用户1的上海节点"
}'
```

### 3. 启动 frps (成都区域)

```toml
# frps-chengdu.toml
bindPort = 7000

[etcd]
endpoints = ["127.0.0.1:2379"]
region = "chengdu"
```

```bash
./frps -c frps-chengdu.toml
```

### 4. 启动 frps (上海区域)

```toml
# frps-shanghai.toml
bindPort = 7000

[etcd]
endpoints = ["127.0.0.1:2379"]
region = "shanghai"
```

```bash
./frps -c frps-shanghai.toml
```

### 5. 客户端连接

```toml
# frpc.toml - 连接成都节点
serverAddr = "chengdu.frps.example.com"
serverPort = 7000
auth.token = "user1_chengdu"

[[proxies]]
name = "web"
type = "tcp"
localPort = 80
remotePort = 8080  # 在允许的端口范围内
```

## 认证流程

1. frpc 使用 token 计算认证密钥连接 frps
2. frps 从 etcd 缓存中查找匹配的 token
3. 验证 token 的 region 是否与 frps 配置的 region 匹配
4. 验证 token 是否启用 (enabled = true)
5. 认证成功后，后续代理请求会：
   - 检查端口是否在 allowPorts 范围内
   - 应用 bandwidthLimit 限速
   - 检查 maxPortsPerClient 限制

## 动态更新

etcd 中的 token 配置支持动态更新：
- 添加新 token：立即生效
- 修改现有 token：立即生效（已连接的客户端在下次创建代理时生效）
- 删除 token：**立即断开已有连接**，新连接会被拒绝
- 禁用 token (enabled=false)：**立即断开已有连接**

## 流量上报

当配置了 `trafficReportUrl` 和 token 的 `trafficReportIntervalMB` 时，frps 会定期向指定 URL 发送流量使用报告。

### 上报格式

POST 请求，Content-Type: application/json

```json
{
  "token": "abc123xyz",
  "region": "chengdu",
  "proxyName": "web",
  "trafficIn": 52428800,
  "trafficOut": 10485760,
  "timestamp": 1707412800
}
```

| 字段 | 说明 |
|------|------|
| `token` | 使用的 token |
| `region` | frps 所在区域 |
| `proxyName` | 代理名称 |
| `trafficIn` | 入站流量（字节） |
| `trafficOut` | 出站流量（字节） |
| `timestamp` | Unix 时间戳 |

### 上报时机

- 当累计流量达到 `trafficReportIntervalMB` 指定的阈值时上报
- 连接关闭时，上报剩余未报告的流量

## 注意事项

1. **region 必须匹配**：token 的 region 必须与 frps 的 etcd.region 完全匹配
2. **端口检查**：只对 TCP 和 UDP 代理检查端口限制，HTTP/HTTPS 等虚拟主机代理不检查
3. **限速模式**：token 的 bandwidthLimit 在服务端生效，需要代理配置 `transport.bandwidthLimitMode = "server"`
4. **优先级**：token 的配置优先于 frps 全局配置和 frpc 代理配置
