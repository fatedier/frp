### SSH Tunnel Gateway

*Added in v0.53.0*

### Concept

SSH supports reverse proxy capabilities [rfc](https://www.rfc-editor.org/rfc/rfc4254#page-16).

frp supports listening on an SSH port on the frps side to achieve TCP protocol proxying using the SSH -R protocol. This mode does not rely on frpc.

SSH reverse tunneling proxying and proxying SSH ports through frp are two different concepts. SSH reverse tunneling proxying is essentially a basic reverse proxying accomplished by connecting to frps via an SSH client when you don't want to use frpc.

```toml
# frps.toml
sshTunnelGateway.bindPort = 0
sshTunnelGateway.privateKeyFile = ""
sshTunnelGateway.autoGenPrivateKeyPath = ""
sshTunnelGateway.authorizedKeysFile = ""
```

| Field | Type | Description | Required |
| :--- | :--- | :--- | :--- |
| bindPort| int | The ssh server port that frps listens on.| Yes |
| privateKeyFile | string | Default value is empty. The private key file used by the ssh server. If it is empty, frps will read the private key file under the autoGenPrivateKeyPath path. It can reuse the /home/user/.ssh/id_rsa file on the local machine, or a custom path can be specified.| No |
| autoGenPrivateKeyPath  | string |Default value is ./.autogen_ssh_key. If the file does not exist or its content is empty, frps will automatically generate RSA private key file content and store it in this file.|No|
| authorizedKeysFile  | string |Default value is empty. If it is empty, ssh client authentication is not authenticated. If it is not empty, it can implement ssh password-free login authentication. It can reuse the local /home/user/.ssh/authorized_keys file or a custom path can be specified.| No |

### Basic Usage

#### Server-side frps

Minimal configuration:

```toml
sshTunnelGateway.bindPort = 2200
```

Place the above configuration in frps.toml and run `./frps -c frps.toml`. It will listen on port 2200 and accept SSH reverse proxy requests.

Note:

1. When using the minimal configuration, a `.autogen_ssh_key` private key file will be automatically created in the current working directory. The SSH server of frps will use this private key file for encryption and decryption. Alternatively, you can reuse an existing private key file on your local machine, such as `/home/user/.ssh/id_rsa`.

2. When running frps in the minimal configuration mode, connecting to frps via SSH does not require authentication. It is strongly recommended to configure a token in frps and specify the token in the SSH command line.

#### Client-side SSH

The command format is:

```bash
ssh -R :80:{local_ip:port} v0@{frps_address} -p {frps_ssh_listen_port} {tcp|http|https|stcp|tcpmux} --remote_port {real_remote_port} --proxy_name {proxy_name} --token {frp_token}
```

1. `--proxy_name` is optional, and if left empty, a random one will be generated.
2. The username for logging in to frps is always "v0" and currently has no significance, i.e., `v0@{frps_address}`.
3. The server-side proxy listens on the port determined by `--remote_port`.
4. `{tcp|http|https|stcp|tcpmux}` supports the complete command parameters, which can be obtained by using `--help`. For example: `ssh -R :80::8080 v0@127.0.0.1 -p 2200 http --help`.
5. The token is optional, but for security reasons, it is strongly recommended to configure the token in frps.

#### TCP Proxy

```bash
ssh -R :80:127.0.0.1:8080 v0@{frp_address} -p 2200 tcp --proxy_name "test-tcp" --remote_port 9090
```

This sets up a proxy on frps that listens on port 9090 and proxies local service on port 8080.

```bash
frp (via SSH) (Ctrl+C to quit)

User: 
ProxyName: test-tcp
Type: tcp
RemoteAddress: :9090
```

Equivalent to:

```bash
frpc tcp --proxy_name "test-tcp" --local_ip 127.0.0.1 --local_port 8080 --remote_port 9090
```

More parameters can be obtained by executing `--help`.

#### HTTP Proxy

```bash
ssh -R :80:127.0.0.1:8080 v0@{frp address} -p 2200 http --proxy_name "test-http"  --custom_domain test-http.frps.com
```

Equivalent to:
```bash
frpc http --proxy_name "test-http" --custom_domain test-http.frps.com
```

You can access the HTTP service using the following command:

curl 'http://test-http.frps.com'

More parameters can be obtained by executing --help.

#### HTTPS/STCP/TCPMUX Proxy

To obtain the usage instructions, use the following command:

```bash
ssh -R :80:127.0.0.1:8080 v0@{frp address} -p 2200 {https|stcp|tcpmux} --help
```

### Advanced Usage

#### Reusing the id_rsa File on the Local Machine

```toml
# frps.toml
sshTunnelGateway.bindPort = 2200
sshTunnelGateway.privateKeyFile = "/home/user/.ssh/id_rsa"
```

During the SSH protocol handshake, public keys are exchanged for data encryption. Therefore, the SSH server on the frps side needs to specify a private key file, which can be reused from an existing file on the local machine. If the privateKeyFile field is empty, frps will automatically create an RSA private key file.

#### Specifying the Auto-Generated Private Key File Path

```toml
# frps.toml
sshTunnelGateway.bindPort = 2200
sshTunnelGateway.autoGenPrivateKeyPath = "/var/frp/ssh-private-key-file"
```

frps will automatically create a private key file and store it at the specified path.

Note: Changing the private key file in frps can cause SSH client login failures. If you need to log in successfully, you can delete the old records from the `/home/user/.ssh/known_hosts` file.

#### Using an Existing authorized_keys File for SSH Public Key Authentication

```toml
# frps.toml
sshTunnelGateway.bindPort = 2200
sshTunnelGateway.authorizedKeysFile = "/home/user/.ssh/authorized_keys"
```

The authorizedKeysFile is the file used for SSH public key authentication, which contains the public key information for users, with one key per line.

If authorizedKeysFile is empty, frps won't perform any authentication for SSH clients. Frps does not support SSH username and password authentication.

You can reuse an existing `authorized_keys` file on your local machine for client authentication.

Note: authorizedKeysFile is for user authentication during the SSH login phase, while the token is for frps authentication. These two authentication methods are independent. SSH authentication comes first, followed by frps token authentication. It is strongly recommended to enable at least one of them. If authorizedKeysFile is empty, it is highly recommended to enable token authentication in frps to avoid security risks.

#### Using a Custom authorized_keys File for SSH Public Key Authentication

```toml
# frps.toml
sshTunnelGateway.bindPort = 2200
sshTunnelGateway.authorizedKeysFile = "/var/frps/custom_authorized_keys_file"
```

Specify the path to a custom `authorized_keys` file.

Note that changes to the authorizedKeysFile file may result in SSH authentication failures. You may need to re-add the public key information to the authorizedKeysFile.
