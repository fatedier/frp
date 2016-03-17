# Quick Start

frp is easier to use compared with other similar projects.

We will use a simple demo to demonstrate how to create a connection to server A's ssh port by server B with public IP address x.x.x.x(replace to the real IP address of your server).

### Download SourceCode

`go get github.com/fatedier/frp` is recommended, then the code will be copied to the directory `$GOPATH/src/github.com/fatedier/frp`.

Or you can use `git clone https://github.com/fatedier/frp.git $GOPATH/src/github.com/fatedier/frp`.

### Compile

Enter the root directory and execute `make`, then wait until finished.

**bin** include all executable programs when **conf** include corresponding configuration files.

### Pre-requirement

* Go environment. Version of go >= 1.4.
* Godep (if not exist, go get will be executed to download godep when compiling)

### Deploy

1. Move `./bin/frps` and `./conf/frps.ini` to any directory of server B.
2. Move `./bin/frpc` and `./conf/frpc.ini` to any directory of server A.
3. Modify all configuration files, details in next paragraph.
4. Execute `nohup ./frps &` or `nohup ./frps -c ./frps.ini &` in server B.
5. Execute `nohup ./frpc &` or `nohup ./frpc -c ./frpc.ini &` in server A.
6. Use `ssh -oPort=6000 {user}@x.x.x.x` to test if frp is work(replace {user} to real username in server A).

### Configuration files

#### frps.ini

```ini
[common]
bind_addr = 0.0.0.0
# for accept connections from frpc
bind_port = 7000
log_file = ./frps.log
log_level = info

# test is the custom name of proxy and there can be many proxies with unique name in one configure file
[test]
passwd = 123
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

# test is proxy name same with configure in frps.ini
[test]
passwd = 123
# local port which need to be transferred
local_port = 22
```
