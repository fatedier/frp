## Run `frp` as Windows service

1. Download [winsw](https://github.com/winsw/winsw/releases)(`WinSW.NET2.exe` means need `.NET 2.0` runtime,and so on),Rename it to `frpc-service.exe` and `frps-service.exe` .
2.  Put `frpc-service.xml` and `frpc-service.xml` to same directory as `frp*-service.exe` .
3. Default location of frp is `C:\frp`, you can edit the xml config file.



install  service 

```shell
# for frp client
frpc-service.exe install 
# for frp server
frps-service.exe install 
```

> You will see frp service in windows service manager



uninstall  service 

```shell
# for frp client
frpc-service.exe uninstall 
# for frp server
frps-service.exe uninstall 
```





## 以Windows服务的方式运行`frp`

1. 下载[winsw](https://github.com/winsw/winsw/releases)(它有多个运行时版本,`WinSW.NET2.exe` 的意思是需要安装 `.NET 2.0` ),然后将其分别重命名为两个文件: `frpc-service.exe` 和`frps-service.exe` .
2.  将`frpc-service.xml` 和 `frpc-service.xml` 放到 `frp*-service.exe` 相同的目录下.
3. 默认的frp安装目录是 `C:\frp`, 你可以在xml配置文件中修改。



创建服务 

```shell
# for frp client
frpc-service.exe install 
# for frp server
frps-service.exe install 
```

> 服务创建后你在Windows的服务管理器里面就能看见frp服务了



服务卸载

```shell
# for frp client
frpc-service.exe uninstall 
# for frp server
frps-service.exe uninstall 
```



