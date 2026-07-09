# frp (bản fork tuỳ biến — MeoBaka)

> Đây là **bản fork tuỳ biến** của frp (fast reverse proxy).
> Toàn bộ tài liệu gốc, kiến trúc, và các tính năng chuẩn của frp xem tại repo chính:
>
> ### 👉 https://github.com/fatedier/frp
>
> README này **chỉ mô tả những phần đã thêm/sửa** so với frp gốc. Mọi thứ khác
> (tcp, udp, http, https, tcpmux, stcp, sudp, xtcp, plugin, dashboard cơ bản, cách
> cài đặt chung…) hoạt động y hệt bản gốc — đọc tài liệu ở link trên.

**Phiên bản fork:** `1.0.0` · **Nhánh:** `meobaka` (base từ frp v0.69.1)

---

## Tóm tắt các thay đổi

| Nhóm | Thay đổi |
|---|---|
| ➕ Type mới | `xudp`, `xtcp+xudp`, `tcp+udp`, `stcp+sudp` |
| 🔒 Bảo mật | Mặc định `transport.wireProtocol` chuyển **v1 → v2** (v1 giữ lại làm tuỳ chọn) |
| 🐛 Sửa lỗi | Kẹt reconnect sau `i/o deadline reached` ở v2 (issue [#5355](https://github.com/fatedier/frp/issues/5355)) |
| 🖥️ Dashboard | frpc admin API + giao diện Vue hỗ trợ đầy đủ các type mới |
| 📄 Ví dụ | `examples/frpc_example.toml`, `examples/frps_example.toml` (chú thích tiếng Việt) |

---

## 1. Type mới

Bốn type dưới đây là **type thật** — đăng ký đầy đủ ở cả frpc lẫn frps, hiện đúng
**1 proxy** trên dashboard (không phải "sugar" tự tách nhiều proxy).

### `xudp` — P2P UDP đục lỗ NAT

Bản UDP của `xtcp`: hai máy tự đục lỗ NAT rồi truyền **UDP** trực tiếp (P2P), không
đi vòng qua frps. Có cả **proxy** (bên cung cấp) và **visitor** (bên truy cập).

```toml
# Bên cung cấp (frpc)
[[proxies]]
name = "game-udp"
type = "xudp"
secretKey = "KEY"
localIP = "127.0.0.1"
localPort = 27015

# Bên truy cập (frpc khác)
[[visitors]]
name = "game-udp-visitor"
type = "xudp"
serverName = "game-udp"
secretKey = "KEY"
bindAddr = "127.0.0.1"
bindPort = 27015
protocol = "quic"     # "quic" (mặc định) hoặc "kcp" — tunnel tin cậy trên lỗ UDP
```

> Khác `xtcp`: `xudp` **không có** `fallbackTo` (UDP mất gói là bình thường).

### `xtcp+xudp` — gộp TCP + UDP qua **1 lỗ NAT** (kiểu tailscale)

1 proxy chở **cả TCP lẫn UDP** tới dịch vụ local qua **cùng một lỗ NAT**. Mỗi
stream trên tunnel được gắn 1 byte tag (0x01 = TCP, 0x02 = UDP) để route. **Hoàn
hảo cho Remote Desktop** (RDP dùng TCP 3389 + UDP 3389 để mượt).

**Có AUTO-FALLBACK sẵn (built-in):** thử P2P đục lỗ trước; nếu quá
`fallbackTimeoutMs` (mặc định 1000ms) mà không đục được (NAT khó / symmetric /
CGNAT) thì **tự chuyển sang relay qua frps** (như `stcp+sudp`) — cho **cả TCP lẫn
UDP**, không cần khai thêm visitor. Provider tự đăng ký sẵn đường relay; hole-punch
work-conn được đánh dấu `protocol="nathole"`, còn conn relay dùng chung đúng khung
1-byte-tag nên provider xử lý bằng cùng một handler.

```toml
# Bên cung cấp (máy có RDP)
[[proxies]]
name = "pc-rdp"
type = "xtcp+xudp"
secretKey = "KEY_RDP"
localIP = "127.0.0.1"
localPort = 3389        # dùng cho CẢ TCP lẫn UDP
# localPortUDP = 3389   # (tuỳ chọn) đổi nếu UDP nghe cổng khác; mặc định = localPort

# Bên truy cập
[[visitors]]
name = "pc-rdp-visitor"
type = "xtcp+xudp"
serverName = "pc-rdp"
secretKey = "KEY_RDP"
bindAddr = "127.0.0.1"
bindPort = 13389        # mở mstsc → 127.0.0.1:13389 để vào Desktop
keepTunnelOpen = true
# fallbackTimeoutMs = 1000   # (tuỳ chọn) chờ P2P bao lâu trước khi rớt relay
```

### `tcp+udp` — gộp TCP + UDP trên **1 cổng public**

1 proxy relay mở **cả TCP lẫn UDP** trên **cùng `remotePort`** ở frps. Cơ chế:
work-conn của TCP mang protocol rỗng (mặc định), work-conn của UDP được gắn cờ
`"udp"` để frpc route đúng handler. Không cần visitor (vào thẳng cổng public).

```toml
[[proxies]]
name = "voice"
type = "tcp+udp"
localIP = "127.0.0.1"
localPort = 8000
localPortUDP = 8001    # (tuỳ chọn) cổng UDP local khác; mặc định = localPort
remotePort = 9000     # cổng TCP+UDP public trên frps
```

### `stcp+sudp` — gộp TCP + UDP **bí mật** qua relay

Bản bí mật của `tcp+udp`: không mở cổng public, vào qua **visitor** giữ cùng
`secretKey`. Trên frps giống hệt stcp/sudp (1 secret listener); phần tách TCP/UDP
nằm hoàn toàn ở client bằng 1 byte tag mỗi stream (ngoài lớp mã hoá).

```toml
# Bên cung cấp
[[proxies]]
name = "priv"
type = "stcp+sudp"
secretKey = "KEY_PRIV"
localIP = "127.0.0.1"
localPort = 22
# localPortUDP = 22   # (tuỳ chọn) mặc định = localPort

# Bên truy cập
[[visitors]]
name = "priv-visitor"
type = "stcp+sudp"
serverName = "priv"     # khớp thẳng name proxy
secretKey = "KEY_PRIV"
bindAddr = "127.0.0.1"
bindPort = 6100         # mở CẢ TCP lẫn UDP tại 127.0.0.1:6100
```

**So sánh nhanh 4 type:**

| Type | Đường đi | Cần visitor? | Dùng cho |
|---|---|---|---|
| `xudp` | P2P đục lỗ NAT | ✅ | UDP P2P độ trễ thấp |
| `xtcp+xudp` | P2P đục lỗ NAT (1 lỗ, TCP+UDP), **tự rớt relay khi NAT khó** | ✅ | Remote Desktop P2P (chắc ăn) |
| `tcp+udp` | Relay qua cổng public frps | ❌ | Dịch vụ TCP+UDP có cổng public |
| `stcp+sudp` | Relay bí mật qua frps | ✅ | Dịch vụ TCP+UDP riêng tư |

---

## 2. Mặc định `wireProtocol = v2`

`transport.wireProtocol` mặc định chuyển từ **v1 → v2** vì v1 bảo mật kém. v2 dùng
codec mới có mã hoá AEAD. Bản v1 của tác giả **giữ nguyên**, vẫn bật lại được:

```toml
[transport]
wireProtocol = "v1"   # ép về v1 nếu cần tương thích ngược
```

Không khai gì → chạy v2. frps tự nhận diện v1/v2 qua magic nên hai bên không cần
khớp thủ công.

---

## 3. Sửa lỗi kẹt reconnect ở v2 (issue #5355)

Triệu chứng gốc: *"出现 i/o deadline reached 后大概率不能重连"* — sau khi gặp
`i/o deadline reached`, frpc login lại bằng `runID` cũ nhưng cứ fail mãi, phải tắt
mở lại frpc. Bản fork **reset `runID` sau 3 lần login thất bại liên tiếp** để lần
sau là một phiên login mới sạch (giống như khởi động lại), giúp tự hồi phục.

---

## 4. Dashboard

frpc admin API và giao diện Vue (`web/frpc`) đã cập nhật để tạo/sửa/hiển thị các
type mới (`xudp`, `xtcp+xudp`, `tcp+udp`, `stcp+sudp`) như proxy/visitor thật.

---

## 5. Build

```bash
# Build nhanh cho dev (bỏ qua nhúng web):
go build -tags noweb -o bin/frpc.exe ./cmd/frpc
go build -tags noweb -o bin/frps.exe ./cmd/frps

# Build đầy đủ có dashboard: build web trước rồi build Go:
cd web/frpc && npm install && npm run build && cd ../..
go build -o bin/frpc.exe ./cmd/frpc
go build -o bin/frps.exe ./cmd/frps
```

Binary xuất ra thư mục `bin/`. Kiểm tra cấu hình mẫu:

```bash
./bin/frpc.exe verify -c examples/frpc_example.toml
./bin/frps.exe verify -c examples/frps_example.toml
```

---

## 6. Ví dụ cấu hình

`examples/frpc_example.toml` và `examples/frps_example.toml` là bộ cấu hình đầy đủ,
chú thích tiếng Việt từng mục — bao gồm toàn bộ type mới ở trên.

---

*Mọi tính năng, tuỳ chọn và tài liệu khác không liệt kê ở đây: xem repo gốc
[fatedier/frp](https://github.com/fatedier/frp).*
