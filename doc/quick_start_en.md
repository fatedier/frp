# Quick Start

frp is easier to use compared with other similar projects.

We will use two simple demo to demonstrate how to use frp.

1. How to create a connection to **server A**'s **ssh port** by **server B** with **public IP address** x.x.x.x(replace to the real IP address of your server).
2. How to visit web service in **server A**'s **8000 port** and **8001 port** by **web01.yourdomain.com** and **web02.yourdomain.com** through **server B** with public ID address.

### Download SourceCode

`go get github.com/fatedier/frp` is recommended, then the code will be copied to the directory `$GOPATH/src/github.com/fatedier/frp`.

Or you can use `git clone https://github.com/fatedier/frp.git $GOPATH/src/github.com/fatedier/frp`.

If you want to try it quickly, download the compiled program and configuration files from [https://github.com/fatedier/frp/releases](https://github.com/fatedier/frp/releases).

### Compile

Enter the root directory and execute `make`, then wait until finished.

**bin** include all executable programs when **conf** include corresponding configuration files.

### Pre-requirement

* Go environment. Version of go >= 1.4.
* Godep (if not exist, `go get` will be executed to download godep when compiling)

### Deploy

1. Move `./bin/frps` and `./conf/frps.ini` to any directory of **server B**.
2. Move `./bin/frpc` and `./conf/frpc.ini` to any directory of **server A**.
3. Modify all configuration files, details in next paragraph.
4. Execute `nohup ./frps &` or `nohup ./frps -c ./frps.ini &` in **server B**.
5. Execute `nohup ./frpc &` or `nohup ./frpc -c ./frpc.ini &` in **server A**.
6. Use `ssh -oPort=6000 {user}@x.x.x.x` to test if frp is work(replace {user} to real username in **server A**), or visit custom domains by browser.

## Tcp port forwarding

### Configuration files

#### frps.ini

```ini
[common]
bind_addr = 0.0.0.0
# for accept connections from frpc
bind_port = 7000
log_file = ./frps.log
log_level = info

# ssh is the custom name of proxy and there can be many proxies with unique name in one configure file
[ssh]
auth_token = 123
bind_addr = 0.0.0.0
# finally we connect to server A by this port
listen_port = 6000
```

#### frpc.ini

```ini
[common]
# server address of frps
server_addr = x.x.x.x
server_port = 7000
log_file = ./frpc.log
log_level = info
# for authentication
auth_token = 123

# ssh is proxy name same with configure in frps.ini
[ssh]
# local port which need to be transferred
local_port = 22
# if use_encryption equals true, messages between frpc and frps will be encrypted, default is false
use_encryption = true
```

## Http port forwarding and Custom domains binding

If you only want to forward port one by one, you just need refer to [Tcp port forwarding](/doc/quick_start_en.md#Tcp-port-forwarding).If you want to visit different web pages deployed in different web servers by **server B**'s **80 port**, you should specify the type as **http**.

You also need to resolve your **A record** of your custom domain to [server_addr], or resolve your **CNAME record** to [server_addr] if [server_addr] is a domain.

After that, you can visit your web pages in local server by custom domains.

### Configuration files

#### frps.ini

```ini
[common]
bind_addr = 0.0.0.0
bind_port = 7000
# if you want to support vhost, specify one port for http services
vhost_http_port = 80
log_file = ./frps.log
log_level = info

[web01]
type = http
auth_token = 123
# # if proxy type equals http, custom_domains must be set separated by commas
custom_domains = web01.yourdomain.com

[web02]
type = http
auth_token = 123
custom_domains = web02.yourdomain.com
```

#### frpc.ini

```ini
[common]
server_addr = x.x.x.x
server_port = 7000
log_file = ./frpc.log
log_level = info
auth_token = 123 

# custom domains are set in frps.ini
[web01]
type = http
local_ip = 127.0.0.1
local_port = 8000
# encryption is optional, default is false
use_encryption = true

[web02]
type = http
local_ip = 127.0.0.1
local_port = 8001
```
