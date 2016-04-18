# frp 使用文档

frp 相比于其他项目而言非常易于部署和使用，这里我们用一个简单的示例演示如何通过一台拥有公网IP地址的服务器B，访问处于内网环境中的服务器A的ssh端口，服务器B的IP地址为 x.x.x.x（测试时替换为真实的IP地址）。

### 下载源码

推荐直接使用 `go get github.com/fatedier/frp` 下载源代码安装，执行命令后代码将会拷贝到 `$GOPATH/src/github.com/fatedier/frp` 目录下。

或者可以使用 `git clone https://github.com/fatedier/frp.git $GOPATH/src/github.com/fatedier/frp` 拷贝到相应目录下。

### 编译

进入下载后的源码根目录，执行 `make` 命令，等待编译完成。

编译完成后， **bin** 目录下是编译好的可执行文件，**conf** 目录下是示例配置文件。

### 依赖

* go 1.4 以上版本
* godep （如果检查不存在，编译时会通过 go get 命令安装）

### 部署

1. 将 ./bin/frps 和 ./conf/frps.ini 拷贝至服务器B任意目录。
2. 将 ./bin/frpc 和 ./conf/frpc.ini 拷贝至服务器A任意目录。
3. 修改两边的配置文件，见下一节说明。
4. 在服务器B执行 `nohup ./frps &` 或者 `nohup ./frps -c ./frps.ini &`。
5. 在服务器A执行 `nohup ./frpc &` 或者 `nohup ./frpc -c ./frpc.ini &`。
6. 通过 `ssh -oPort=6000 {user}@x.x.x.x` 测试是否能够成功连接服务器A（{user}替换为服务器A上存在的真实用户）。

### 配置文件

#### frps.ini

```ini
[common]
bind_addr = 0.0.0.0
# 用于接收 frpc 连接的端口
bind_port = 7000
log_file = ./frps.log
log_level = info

# test 为代理的自定义名称，可以有多个，不能重复，和frpc中名称对应
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

# test需要和 frps.ini 中配置一致
[ssh]
# 需要转发的本地端口
local_port = 22
# 启用加密，frpc与frps之间通信加密，默认为 false
use_encryption = true
```
