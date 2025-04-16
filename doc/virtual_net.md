# Virtual Network (VirtualNet)

*Alpha feature added in v0.62.0*

The VirtualNet feature enables frp to create and manage virtual network connections between clients and visitors through a TUN interface. This allows for IP-level routing between machines, extending frp beyond simple port forwarding to support full network connectivity.

> **Note**: VirtualNet is an Alpha stage feature and is currently unstable. Its configuration methods and functionality may be adjusted and changed at any time in subsequent versions. Do not use this feature in production environments; it is only recommended for testing and evaluation purposes.

## Enabling VirtualNet

Since VirtualNet is currently an alpha feature, you need to enable it with feature gates in your configuration:

```toml
# frpc.toml
featureGates = { VirtualNet = true }
```

## Basic Configuration

To use the virtual network capabilities:

1. First, configure your frpc with a virtual network address:

```toml
# frpc.toml
serverAddr = "x.x.x.x"
serverPort = 7000
featureGates = { VirtualNet = true }

# Configure the virtual network interface
virtualNet.address = "100.86.0.1/24"
```

2. For client proxies, use the `virtual_net` plugin:

```toml
# frpc.toml (server side)
[[proxies]]
name = "vnet-server"
type = "stcp"
secretKey = "your-secret-key"
[proxies.plugin]
type = "virtual_net"
```

3. For visitor connections, configure the `virtual_net` visitor plugin:

```toml
# frpc.toml (client side)
serverAddr = "x.x.x.x"
serverPort = 7000
featureGates = { VirtualNet = true }

# Configure the virtual network interface
virtualNet.address = "100.86.0.2/24"

[[visitors]]
name = "vnet-visitor"
type = "stcp"
serverName = "vnet-server"
secretKey = "your-secret-key"
bindPort = -1
[visitors.plugin]
type = "virtual_net"
destinationIP = "100.86.0.1"
```

## Requirements and Limitations

- **Permissions**: Creating a TUN interface requires elevated permissions (root/admin)
- **Platform Support**: Currently supported on Linux and macOS
- **Default Status**: As an alpha feature, VirtualNet is disabled by default
- **Configuration**: A valid IP/CIDR must be provided for each endpoint in the virtual network 
