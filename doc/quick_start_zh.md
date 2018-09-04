# frp 使用文档

相比于其他项目而言 frp 更易于部署和使用，这里我们用两个简单的示例来演示 frp 的使用过程。

1. 如何通过一台拥有公网IP地址的**服务器B**，访问处于公司内部网络环境中的**服务器A**的**ssh**端口，**服务器B**的IP地址为 x.x.x.x（测试时替换为真实的IP地址）。
2. 如何利用一台拥有公网IP地址的**服务器B**，使通过 **web01.yourdomain.com** 可以访问内网环境中**服务器A**上**8000端口**的web服务，**web02.yourdomain.com** 可以访问**服务器A**上**8001端口**的web服务。

### 下载源码

推荐直接使用 `go get github.com/fatedier/frp` 下载源代码安装，执行命令后代码将会拷贝到 `$GOPATH/src/github.com/fatedier/frp` 目录下。

或者可以使用 `git clone https://github.com/fatedier/frp.git $GOPATH/src/github.com/fatedier/frp` 拷贝到相应目录下。

如果您想快速进行测试，也可以根据您服务器的操作系统及架构直接下载编译好的程序及示例配置文件，[https://github.com/fatedier/frp/releases](https://github.com/fatedier/frp/releases)。

### 编译

进入下载后的源码根目录，执行 `make` 命令，等待编译完成。

编译完成后， **bin** 目录下是编译好的可执行文件，**conf** 目录下是示例配置文件。

### 依赖

* go 1.4 以上版本
* godep （如果检查不存在，编译时会通过 `go get` 命令安装）

### 部署

1. 将 ./bin/frps 和 ./conf/frps.ini 拷贝至**服务器B**任意目录。
2. 将 ./bin/frpc 和 ./conf/frpc.ini 拷贝至**服务器A**任意目录。
3. 根据要实现的功能修改两边的配置文件，详细内容见后续章节说明。
4. 在服务器B执行 `nohup ./frps &` 或者 `nohup ./frps -c ./frps.ini &`。
5. 在服务器A执行 `nohup ./frpc &` 或者 `nohup ./frpc -c ./frpc.ini &`。
6. 通过 `ssh -oPort=6000 {user}@x.x.x.x` 测试是否能够成功连接**服务器A**（{user}替换为**服务器A**上存在的真实用户），或通过浏览器访问自定义域名验证 http 服务是否转发成功。

## tcp 端口转发

转发 tcp 端口需要按照需求修改 frps 和 frpc 的配置文件。

### 配置文件

#### frps.ini

```ini
[common]
bind_addr = 0.0.0.0
# 用于接收 frpc 连接的端口
bind_port = 7000
log_file = ./frps.log
log_level = info

# ssh 为代理的自定义名称，可以有多个，不能重复，和frpc中名称对应
[ssh]
auth_token = 123 
bind_addr = 0.0.0.0
# 最后将通过此端口访问后端服务
listen_port = 6000
```

#### frpc.ini

```ini
[common]
# frps 所在服务器绑定的IP地址
server_addr = x.x.x.x
server_port = 7000
log_file = ./frpc.log
log_level = info
# 用于身份验证
auth_token = 123 

# ssh 需要和 frps.ini 中配置一致
[ssh]
# 需要转发的本地端口
local_port = 22
# 启用加密，frpc与frps之间通信加密，默认为 false
use_encryption = true
```

## http 端口转发，自定义域名绑定

如果只需要一对一的转发，例如**服务器B**的**80端口**转发**服务器A**的**8000端口**，则只需要配置 [tcp 端口转发](/doc/quick_start_zh.md#tcp-端口转发) 即可，如果需要使**服务器B**的**80端口**可以转发至**多个**web服务端口，则需要指定代理的类型为 http，并且在 frps 的配置文件中配置用于提供 http 转发服务的端口。

按照如下的内容修改配置文件后，需要将自定义域名的 **A 记录**解析到 [server_addr]，如果 [server_addr] 是域名也可以将自定义域名的 **CNAME 记录**解析到 [server_addr]。

之后就可以通过自定义域名访问到本地的多个 web 服务。

### 配置文件

#### frps.ini

```ini
[common]
bind_addr = 0.0.0.0
bind_port = 7000
# 如果需要支持http类型的代理则需要指定一个端口
vhost_http_port = 80
log_file = ./frps.log
log_level = info

[web01]
# type 默认为 tcp，这里需要特别指定为 http
type = http
auth_token = 123
# 自定义域名绑定，如果需要同时绑定多个以英文逗号分隔
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


# 自定义域名在 frps.ini 中配置，方便做统一管理
[web01]
type = http
local_ip = 127.0.0.1
local_port = 8000
# 可选是否加密
use_encryption = true

[web02]
type = http
local_ip = 127.0.0.1
local_port = 8001
```
