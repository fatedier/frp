# frp

[![Build Status](https://travis-ci.org/fatedier/frp.svg?branch=master)](https://travis-ci.org/fatedier/frp)
[![GitHub release](https://img.shields.io/github/tag/fatedier/frp.svg?label=release)](https://github.com/fatedier/frp/releases)

[README](README.md) | [中文文档](README_zh.md)

frp 是一个专注于内网穿透的高性能的反向代理应用，支持 TCP、UDP、HTTP、HTTPS 等多种协议。可以将内网服务以安全、便捷的方式通过具有公网 IP 节点的中转暴露到公网。

> 该版本在原版Dashboard基础上增加了一个自动读取TCP代理的导航页面。 该页面利用了不常用的空置属性`group_key`和`group`来传递直连地址和导航图片，对于非HTTP链接可以可以不用配置这部分内容(如SSH)，这种情况下导航页面会忽略该代理项。使用`group`字段配置导航图标可以是BASE64格式也可以是可链接地URL，配置示例：
```
[Nextcloud]
type = tcp
local_ip = 127.0.0.1
local_port = 3000
remote_port = 6000
group = data:image/svg+xml;base64,PHN2ZyB0PSIxNjgxOTA4ODAyODYxIiBjbGFzcz0iaWNvbiIgdmlld0JveD0iMCAwIDEwMjQgMTAyNCIgdmVyc2lvbj0iMS4xIiB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHAtaWQ9IjEyMDgiIHdpZHRoPSIyMDAiIGhlaWdodD0iMjAwIj48cGF0aCBkPSJNMTAyLjQgMTAyLjRtMzA3LjIgMGwyMDQuOCAwcTMwNy4yIDAgMzA3LjIgMzA3LjJsMCAyMDQuOHEwIDMwNy4yLTMwNy4yIDMwNy4ybC0yMDQuOCAwcS0zMDcuMiAwLTMwNy4yLTMwNy4ybDAtMjA0LjhxMC0zMDcuMiAzMDcuMi0zMDcuMloiIGZpbGw9IiM1MENDQ0MiIHAtaWQ9IjEyMDkiPjwvcGF0aD48cGF0aCBkPSJNNTEzLjcwNDk2IDMwNy4yQzU4OS4xMTIzMiAzMDcuMiA2NTAuMjQgMzY4LjMzMjggNjUwLjI0IDQ0My43MzUwNGM3NS40MDczNiAwIDEzNi41MzUwNCA2MS4xMjc2OCAxMzYuNTM1MDQgMTM2LjUyOTkyIDAgNzUuNDA3MzYtNjEuMTMyOCAxMzYuNTM1MDQtMTM2LjUzNTA0IDEzNi41MzUwNEgzNzcuMTc1MDRDMzAxLjc2NzY4IDcxNi44IDI0MC42NCA2NTUuNjY3MiAyNDAuNjQgNTgwLjI2NDk2YzAtNzUuNDAyMjQgNjEuMTMyOC0xMzYuNTI5OTIgMTM2LjUzNTA0LTEzNi41Mjk5MmwwLjAxNTM2LTIuMjU3OTJDMzc4LjM5ODcyIDM2Ny4xMDQgNDM5LjA1NTM2IDMwNy4yIDUxMy43MDQ5NiAzMDcuMnoiIGZpbGw9IiNGRkZGRkYiIGZpbGwtb3BhY2l0eT0iLjgiIHAtaWQ9IjEyMTAiPjwvcGF0aD48L3N2Zz4=
group_key = https://nextcloud.domain.com

```

导航效果图：

![image](https://raw.githubusercontent.com/synebula/frp-with-navigation-page/dev/doc/pic/navigation.png)

<h3 align="center">Gold Sponsors</h3>
<!--gold sponsors start-->
<p align="center">
  <a href="https://workos.com/?utm_campaign=github_repo&utm_medium=referral&utm_content=frp&utm_source=github" target="_blank">
    <img width="350px" src="https://raw.githubusercontent.com/fatedier/frp/dev/doc/pic/sponsor_workos.png">
  </a>
  <a>&nbsp</a>
  <a href="https://asocks.com/c/vDu6Dk" target="_blank">
    <img width="350px" src="https://raw.githubusercontent.com/fatedier/frp/dev/doc/pic/sponsor_asocks.jpg">
  </a>
</p>
<!--gold sponsors end-->

## 为什么使用 frp ？

通过在具有公网 IP 的节点上部署 frp 服务端，可以轻松地将内网服务穿透到公网，同时提供诸多专业的功能特性，这包括：

* 客户端服务端通信支持 TCP、KCP 以及 Websocket 等多种协议。
* 采用 TCP 连接流式复用，在单个连接间承载更多请求，节省连接建立时间。
* 代理组间的负载均衡。
* 端口复用，多个服务通过同一个服务端端口暴露。
* 多个原生支持的客户端插件（静态文件查看，HTTP、SOCK5 代理等），便于独立使用 frp 客户端完成某些工作。
* 高度扩展性的服务端插件系统，方便结合自身需求进行功能扩展。
* 服务端和客户端 UI 页面。

## 开发状态

frp 目前已被很多公司广泛用于测试、生产环境。

master 分支用于发布稳定版本，dev 分支用于开发，您可以尝试下载最新的 release 版本进行测试。

我们正在进行 v2 大版本的开发，将会尝试在各个方面进行重构和升级，且不会与 v1 版本进行兼容，预计会持续一段时间。

现在的 v0 版本将会在合适的时间切换为 v1 版本并且保证兼容性，后续只做 bug 修复和优化，不再进行大的功能性更新。

## 文档

完整文档已经迁移至 [https://gofrp.org](https://gofrp.org/docs)。

## 为 frp 做贡献

frp 是一个免费且开源的项目，我们欢迎任何人为其开发和进步贡献力量。

* 在使用过程中出现任何问题，可以通过 [issues](https://github.com/fatedier/frp/issues) 来反馈。
* Bug 的修复可以直接提交 Pull Request 到 dev 分支。
* 如果是增加新的功能特性，请先创建一个 issue 并做简单描述以及大致的实现方法，提议被采纳后，就可以创建一个实现新特性的 Pull Request。
* 欢迎对说明文档做出改善，帮助更多的人使用 frp，特别是英文文档。
* 贡献代码请提交 PR 至 dev 分支，master 分支仅用于发布稳定可用版本。
* 如果你有任何其他方面的问题或合作，欢迎发送邮件至 fatedier@gmail.com 。

**提醒：和项目相关的问题最好在 [issues](https://github.com/fatedier/frp/issues) 中反馈，这样方便其他有类似问题的人可以快速查找解决方法，并且也避免了我们重复回答一些问题。**

## 捐助

如果您觉得 frp 对你有帮助，欢迎给予我们一定的捐助来维持项目的长期发展。

### GitHub Sponsors

您可以通过 [GitHub Sponsors](https://github.com/sponsors/fatedier) 赞助我们。

企业赞助者可以将贵公司的 Logo 以及链接放置在项目 README 文件中。

### 知识星球

如果您想学习 frp 相关的知识和技术，或者寻求任何帮助及咨询，都可以通过微信扫描下方的二维码付费加入知识星球的官方社群：

![zsxq](/doc/pic/zsxq.jpg)

### 支付宝扫码捐赠

![donate-alipay](/doc/pic/donate-alipay.png)

### 微信支付捐赠

![donate-wechatpay](/doc/pic/donate-wechatpay.png)
