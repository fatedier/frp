# frp

[![Build Status](https://travis-ci.org/fatedier/frp.svg)](https://travis-ci.org/fatedier/frp)

[README](README.md) | [中文文档](README_zh.md)

>frp 是一个高性能的反向代理应用，可以帮助你轻松的进行内网穿透，对外网提供服务，对于 http 服务还支持虚拟主机功能，访问80端口，可以根据域名路由到后端不同的 http 服务。

## frp 的作用?

* 利用处于内网或防火墙后的机器，对外网环境提供 http 服务。
* 对于 http 服务支持基于域名的虚拟主机，支持自定义域名绑定，使多个域名可以共用一个80端口。
* 利用处于内网或防火墙后的机器，对外网环境提供 tcp 服务，例如在家里通过 ssh 访问公司局域网内的主机。
* 可查看通过代理的所有 http 请求和响应的详细信息。（待开发）

## 开发状态

frp 目前正在前期开发阶段，master分支用于发布稳定版本，dev分支用于开发，您可以尝试下载最新的 release 版本进行测试。

**在 1.0 版本以前，交互协议都可能会被改变，不能保证向后兼容。**

## 快速开始

[使用文档](/doc/quick_start_zh.md)

[tcp 端口转发](/doc/quick_start_zh.md#tcp-端口转发)

[http 端口转发，自定义域名绑定](/doc/quick_start_zh.md#http-端口转发自定义域名绑定)

## 架构

![architecture](/doc/pic/architecture.png)

## 贡献代码

如果您对这个项目感兴趣，我们非常欢迎您参与其中！

* 如果您需要提交问题，可以通过 [issues](https://github.com/fatedier/frp/issues) 来完成。
* 如果您有新的功能需求，可以反馈至 fatedier@gmail.com 共同讨论。

## 贡献者

* [fatedier](https://github.com/fatedier)
* [Hurricanezwf](https://github.com/Hurricanezwf)
* [vashstorm](https://github.com/vashstorm)
