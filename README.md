# frp Auto Transport Edition

This repository is based on the original [frp](https://github.com/fatedier/frp) project.

Compared with the original version, this version adds **Auto Transport Mode** for the connection between `frpc` and `frps`.

When both client and server configure:

```toml
[transport]
protocol = "auto"
```

`frpc` can automatically select and fail over between supported transport protocols, including:

- `tcp`
- `kcp`
- `quic`
- `websocket`
- `wss`

The selection is client-driven. `frps` only advertises available transport endpoints and validates the final choice.

For other frp features, usage, and documentation, please refer to the original project:

- GitHub: <https://github.com/fatedier/frp>
- Documentation: <https://gofrp.org>

For the Auto Transport Mode design details in this repository, see:

- [doc/auto_transport.md](./doc/auto_transport.md)
