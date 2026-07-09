# Plan: XUDP + Combo Types cho fork `MeoBaka/frp`

> Trạng thái: **BẢN NHÁP CHỜ DUYỆT** — chưa động vào code.
> Mục tiêu: thêm P2P cho UDP và các "combo type", đặc biệt phục vụ RDP mượt (TCP 3389 + UDP 3389 cùng đi P2P).

---

## 0. Tổng quan & nguyên tắc

Ba tính năng, theo thứ tự phụ thuộc:

| # | Tính năng | Bản chất | Độ khó |
|---|-----------|----------|--------|
| A | **`xudp`** (P2P UDP độc lập) | Tính năng core mới | 🔴 Cao |
| B | **`xtcp+xudp`** (1 lỗ chung chở cả TCP+UDP) | Multiplex 2 loại trên 1 session | 🔴 Cao |
| C | **Combo sugar** (`tcp+udp`, `http+https`, `stcp+sudp`) | Đường cú pháp, tách 1→N | 🟢 Thấp |

**Nguyên tắc vàng (từ khảo sát source):** cái lỗ NAT đã là UDP sẵn, và frp đã có sẵn toàn bộ code chở UDP (`sudp`/`udp`). Nên KHÔNG viết lại gì về đục lỗ hay đóng gói UDP — chỉ **ghép** bộ xương `xtcp` (đục lỗ + QUIC/KCP session) với ruột `sudp` (khung `msg.UDPPacket`).

### Thành phần TÁI DÙNG NGUYÊN (không đụng)

| Thành phần | Vị trí | Vai trò |
|-----------|--------|---------|
| `nathole.Prepare/ExchangeInfo/MakeHole/PreCheck` | pkg/nathole/* | Đục lỗ — trả về `*net.UDPConn` |
| `TunnelSession` + `QUICTunnelSession` + `KCPTunnelSession` | client/visitor/xtcp.go:321-436 | Kênh tin cậy đa-stream trên lỗ UDP |
| `msg.UDPPacket` | pkg/msg/msg.go:186 | Đơn vị đóng gói 1 datagram |
| `udp.NewUDPPacket` / `udp.GetContent` | pkg/proto/udp/udp.go:29,39 | datagram ⇄ struct |
| `udp.ForwardUserConn` | pkg/proto/udp/udp.go:43 | Vòng lặp socket UDP local ↔ channel |
| `udp.Forwarder` | pkg/proto/udp/udp.go:73 | Dial tới service UDP local ↔ channel |
| `msg.NewConn` / `msg.NewReadWriter` | pkg/msg/handler.go:37,79 | Codec length-prefixed trên stream |
| Server nathole signaling | server/proxy/xtcp.go, server/control.go, pkg/nathole/controller.go | **frps không phân biệt TCP/UDP** → 0 thay đổi hành vi |

---

## PHASE A — `xudp` độc lập

### A.1 Config types

**File `pkg/config/v1/proxy.go`:**
- Thêm hằng cạnh dòng 237: `ProxyTypeXUDP ProxyType = "xudp"`
- Thêm vào `proxyConfigTypeMap` (dòng 241-250): `ProxyTypeXUDP: reflect.TypeFor[XUDPProxyConfig]()`
- Clone `XUDPProxyConfig` từ `XTCPProxyConfig` (proxy.go:472-504) — **giống hệt về field** (embeds `ProxyBaseConfig` → có sẵn `LocalIP`/`LocalPort`, `Secretkey`, `AllowUsers`, `NatTraversal`). Chỉ đổi tên type + `MarshalToMsg`/`UnmarshalFromMsg` giữ nguyên logic (chỉ chở `Sk`+`AllowUsers`).

```go
type XUDPProxyConfig struct {
	ProxyBaseConfig
	Secretkey    string   `json:"secretKey,omitempty"`
	AllowUsers   []string `json:"allowUsers,omitempty"`
	NatTraversal *NatTraversalConfig `json:"natTraversal,omitempty"`
}
// Complete/Clone/MarshalToMsg/UnmarshalFromMsg: copy y hệt XTCP
```

**File `pkg/config/v1/visitor.go`:**
- Hằng cạnh 77: `VisitorTypeXUDP VisitorType = "xudp"`
- `visitorConfigTypeMap` (81-85): thêm entry
- Clone `XUDPVisitorConfig` từ `XTCPVisitorConfig` (visitor.go:143-171). **Giữ `Protocol` (quic/kcp) cho TUNNEL transport — KHÔNG overload nó cho TCP/UDP.** Giữ `KeepTunnelOpen`, `MaxRetriesAnHour`, `MinRetryInterval`, `NatTraversal`.
- ⚠️ **Bỏ hoặc thiết kế lại `FallbackTo`/`FallbackTimeoutMs`** — xem mục A.5.

> Decode tự động chạy qua registry (decode.go:44-70 proxy, :72-98 visitor) — không cần sửa decode.

### A.2 Visitor `xudp` — **file mới `client/visitor/xudp.go`**

Công thức: **copy `client/visitor/sudp.go` gần như y nguyên**, thay nguồn tunnel-conn từ `getNewVisitorConn()` (dial frps) sang **stream mở trên session đục lỗ** (mượn từ xtcp).

Cấu trúc:
```go
type XUDPVisitor struct {
	*BaseVisitor
	cfg      *v1.XUDPVisitorConfig
	udpConn  *net.UDPConn                 // local bindAddr:bindPort  (từ sudp)
	readCh   chan *msg.UDPPacket
	sendCh   chan *msg.UDPPacket
	session  TunnelSession                // từ xtcp (QUIC/KCP)
	// + các field đục lỗ: startTunnelCh, retry... (từ xtcp)
}
```

Luồng `Run()`:
1. Chọn transport theo `cfg.Protocol`: `NewKCPTunnelSession()` / `NewQUICTunnelSession()` — **copy xtcp.go:54-60**.
2. `net.ResolveUDPAddr` + `net.ListenUDP(bindAddr:bindPort)` → `sv.udpConn` — **copy sudp.go:49-57** (KHÔNG phải `net.Listen("tcp")` như xtcp.go:64).
3. `go udp.ForwardUserConn(sv.udpConn, sv.readCh, sv.sendCh, packetSize)` — **copy sudp.go:65** (vòng lặp socket UDP local).
4. `go sv.dispatcher()` — chờ gói đầu tiên → đục lỗ → mở stream → chạy `worker`.

`makeNatHole()` — **copy nguyên xtcp.go:252-317** (Prepare→ExchangeInfo→MakeHole→`session.Init(listenConn, raddr)`).

`worker(tunnelStream net.Conn)` — **copy sudp.go:113-200**:
- `payloadConn := msg.NewConn(tunnelStream, msg.NewReadWriter(tunnelStream, wireProtocol))`
- `workConnReaderFn`: `payloadConn.ReadMsg()` → `*msg.UDPPacket` → `readCh` (tunnel→user)
- `workConnSenderFn`: `sendCh` → `payloadConn.WriteMsg()` (user→tunnel)

**Khác biệt DUY NHẤT so với sudp:** thay `getNewVisitorConn()` (sudp.go:202) bằng `sv.session.OpenConn(ctx)` (stream trên lỗ đục được).

### A.3 Provider `xudp` — **file mới `client/proxy/xudp.go`**

Copy `client/proxy/xtcp.go`, mép local đổi TCP→UDP:
- `InWorkConn` → `nathole.Prepare/ExchangeInfo/MakeHole` → `listenByKCP`/`listenByQUIC`: **copy nguyên xtcp.go:57-215**.
- **Đổi handler mỗi stream:** xtcp.go:176 (`go pxy.HandleTCPWorkConnection(...)`) và :214 → thay bằng forwarder UDP:
  ```go
  go pxy.handleUDPWorkConnection(muxConn)  // thay HandleTCPWorkConnection
  ```
- `handleUDPWorkConnection(stream net.Conn)`: wrap `msg.NewConn`, chạy `udp.Forwarder(localUDPAddr, readCh, sendCh, ...)` với `localUDPAddr = net.ResolveUDPAddr(LocalIP, LocalPort)` — **mẫu client/proxy/sudp.go:79-193** (thay cho `HandleTCPWorkConnection` dùng `libnet.Dial` TCP ở proxy.go:213).
- Đăng ký factory: `RegisterProxyFactory(reflect.TypeOf(&v1.XUDPProxyConfig{}), NewXUDPProxy)` — mẫu xtcp.go:36-38.

### A.4 Server `xudp` — **file mới `server/proxy/xudp.go`**

**Copy y hệt `server/proxy/xtcp.go:27-101`**, chỉ đổi `*v1.XTCPProxyConfig` → `*v1.XUDPProxyConfig`. Không có gì TCP-specific (chỉ `NatHoleController.ListenClient` + `writeNatHoleSid`). **Không đụng** server/control.go hay pkg/nathole/controller.go.

### A.5 Xử lý `FallbackTo` (caveat quan trọng)

xtcp fallback (xtcp.go:161-183) gọi `helper.TransferConn(fallbackName, userConn)` với `userConn` là **TCP**. xudp không có object TCP per-connection để transfer.
- **Quyết định Phase A:** BỎ `FallbackTo` khỏi `XUDPVisitorConfig` (fallback UDP vô nghĩa — UDP mất gói là bình thường). Nếu đục lỗ thất bại → log cảnh báo + retry (`MaxRetriesAnHour`). Ghi rõ trong doc.

### A.6 Plumbing (cơ học, cùng lúc với A.1)

| File | Sửa |
|------|-----|
| pkg/config/v1/validation/proxy.go:108-125 & 174-193 | thêm `case *v1.XUDPProxyConfig` |
| pkg/config/v1/validation/visitor.go:31-38 | thêm case xudp |
| client/visitor/visitor.go:~99 (switch runtime) | `case *v1.XUDPVisitorConfig: return NewXUDPVisitor(...)` |
| cmd/frpc/sub/proxy.go:31-40 (`proxyTypes`), :42-46 (`visitorTypes`) | append `xudp` |
| client/http/model/proxy_definition.go & visitor_definition.go | thêm field + case xudp |
| web/frpc/src/types/constants.ts, proxy-converters.ts | thêm `'xudp'` |
| web/frps/src/utils/proxy.ts | thêm class hiển thị |

### A.7 Test Phase A
- Unit: config load `type="xudp"` + visitor `type="xudp"` decode/validate OK.
- E2E thủ công: provider xudp → `localPort=3389` (hoặc 1 UDP echo test), visitor `bindPort=13389`, bắn UDP qua `127.0.0.1:13389` → thấy về đúng. Dùng đúng hạ tầng VPS 103.110.32.128 hiện có.
- Kiểm tra log đục lỗ thành công (nat hole success).

---

## PHASE B — `xtcp+xudp` gộp 1 lỗ chung (kiểu tailscale)

Mục tiêu: **một** proxy, **một** lần đục lỗ, chở **cả** TCP lẫn UDP qua **cùng** QUIC/KCP session (nhiều stream). Đây là chỗ frp tiến gần L3 nhất.

### B.1 Ý tưởng cốt lõi: gắn thẻ (tag) cho stream

QUIC/yamux đã đa-stream. Ta multiplex 2 loại traffic trên 1 session bằng cách **byte tag đầu mỗi stream**:

```
Khi VISITOR mở 1 stream mới, ghi 1 byte đầu tiên:
  0x01 = stream chở 1 kết nối TCP  (như xtcp hiện tại)
  0x02 = stream chở kênh UDP-packet (như xudp/sudp)
PROVIDER đọc byte đầu -> route:
  0x01 -> dial TCP  localIP:localPort  + libio.Join   (HandleTCPWorkConnection)
  0x02 -> udp.Forwarder localIP:localPort(UDP)
```

Vì cả 2 đầu của type gộp đều là code MỚI → tự do định nghĩa protocol này, không phá vỡ tương thích với xtcp/xudp cũ.

### B.2 Config

Type mới `xtcpxudp` (hoặc giữ tên UI `xtcp+xudp`, map nội bộ về 1 hằng):
```go
ProxyTypeXTCPXUDP ProxyType = "xtcpxudp"
```
- Provider: `LocalIP` + `LocalPort` dùng CHUNG cho cả TCP và UDP (khớp RDP: 3389 cả hai). Tùy chọn `LocalPortUDP` để override.
- Visitor: `BindAddr` + `BindPort` → mở **cả** TCP listener **lẫn** UDP listener trên cùng cổng.

### B.3 Visitor gộp — **file mới `client/visitor/xtcpxudp.go`**

- Đục lỗ 1 lần (`makeNatHole`, copy xtcp) → 1 `session`.
- Mở **2 mặt tiền local song song**:
  - TCP listener (copy xtcp.go:63-68 + acceptLoop): mỗi conn → `session.OpenConn` → **ghi tag 0x01** → `libio.Join`.
  - UDP listener (copy A.2): 1 stream → `session.OpenConn` → **ghi tag 0x02** → pump `msg.UDPPacket`.

### B.4 Provider gộp — **file mới `client/proxy/xtcpxudp.go`**

- Đục lỗ + `listenByKCP/QUIC` (copy xtcp) → vòng `Accept()` stream.
- Mỗi stream: **đọc 1 byte tag** → route:
  - `0x01` → `HandleTCPWorkConnection` (TCP dial, y hệt xtcp)
  - `0x02` → `handleUDPWorkConnection` (UDP forwarder, từ A.3)

### B.5 Server gộp
- Copy `server/proxy/xtcp.go` → `server/proxy/xtcpxudp.go`, đổi type. Vẫn 0 thay đổi hành vi frps.

### B.6 Test Phase B
- RDP thật: provider `xtcpxudp localPort=3389`, visitor `bindPort=13389`. mstsc `127.0.0.1:13389` → vào Desktop, và kênh UDP RemoteFX cũng đi qua (kiểm bằng `netstat`/perfmon trong phiên RDP thấy dùng UDP).
- Xác nhận CHỈ đục 1 lỗ (1 dòng "nat hole success" thay vì 2).

---

## PHASE C — Combo sugar (`tcp+udp`, `http+https`, `stcp+sudp`)

Thuần "đường cú pháp": 1 entry config → nhiều proxy con. KHÔNG đụng runtime/protocol.

### C.1 Decode được combo
**File `pkg/config/v1/proxy.go`:** thêm `ComboProxyConfig` + đăng ký vào `proxyConfigTypeMap` (nếu không, decode.go:55-57 sẽ báo `unknown proxy type`).
```go
type ComboProxyConfig struct {
	ProxyBaseConfig            // name, type, localIP, localPort, sk...
	// giữ đủ field union để sinh con; hoặc lưu raw rồi tách
}
```
Hằng: `ProxyTypeTCPUDP="tcp+udp"`, `ProxyTypeHTTPHTTPS="http+https"`, `ProxyTypeSTCPSUDP="stcp+sudp"`, `ProxyTypeXTCPXUDP` (nếu muốn combo này cũng theo đường sugar — nhưng ta đã làm 1-lỗ ở Phase B, nên `xtcp+xudp` trỏ về type gộp B, KHÔNG expand).

### C.2 Pass mở rộng
**File `pkg/config/load.go`, chèn tại dòng ~412** (trong `LoadClientConfig`, ngay trước `CompleteProxyConfigurers`/`FilterClientConfigurers`):
```go
proxyCfgs = ExpandComboProxyConfigurers(proxyCfgs)   // 1 combo -> N proxy con
```
Hàm mới `ExpandComboProxyConfigurers([]v1.ProxyConfigurer) []v1.ProxyConfigurer`:
- Duyệt; nếu là `*ComboProxyConfig` → sinh các con theo quy ước tên:
  - `name="rdp", type="tcp+udp"` → `rdp-tcp` (TCPProxyConfig) + `rdp-udp` (UDPProxyConfig), copy localIP/localPort/... sang cả hai.
  - `http+https`, `stcp+sudp`: tương tự.
- Đặt trước Complete/Filter để con được `Complete()` + lọc start/enabled như thường.

### C.3 Quy ước đặt tên con
`<name>-<subtype>` (vd `rdp-tcp`, `rdp-udp`). Ghi rõ trong docs để tránh trùng tên.

### C.4 Test Phase C
- `type="tcp+udp"` → sau load thấy đúng 2 proxy `-tcp` và `-udp`.
- Validate/complete chạy đúng trên con.

---

## Thứ tự thực hiện & mốc kiểm

```
1. Phase A.1 + A.6  (config + plumbing xudp)        -> build pass
2. Phase A.2-A.4    (visitor/provider/server xudp)  -> test UDP echo P2P
3. Phase A.7        (E2E xudp)                       -> ✅ mốc 1: UDP đi P2P
4. Phase B          (xtcp+xudp 1 lỗ)                 -> ✅ mốc 2: RDP full P2P
5. Phase C          (combo sugar)                    -> ✅ mốc 3: khai gọn
```

## Rủi ro & lưu ý

| Rủi ro | Xử lý |
|--------|-------|
| `FallbackTo` không áp dụng cho UDP | Bỏ ở xudp (A.5); xtcpxudp giữ fallback cho phần TCP nếu cần |
| `ForwardUserConn`/`Forwarder` drop gói khi channel đầy | **Giữ nguyên** — đúng semantics UDP (udp.go:66-69) |
| Đục lỗ thất bại (CGNAT/symmetric) | Retry theo `MaxRetriesAnHour`; xudp KHÔNG có relay-fallback → log rõ |
| Web UI / admin API nhiều chỗ hardcode type | Đã liệt kê ở A.6; làm cuối, không chặn core |
| Tag stream (Phase B) sai đồng bộ | Viết test 2 đầu; tag là byte đầu tiên, đọc trước khi route |

## Ước lượng khối lượng
- Phase A: ~4 file mới + ~8 file sửa nhẹ. Phần lớn là copy-adapt.
- Phase B: ~3 file mới (dựa trên A).
- Phase C: ~2 file sửa.
