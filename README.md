# frp

[![Build Status](https://travis-ci.org/fatedier/frp.svg)](https://travis-ci.org/fatedier/frp)

## What is frp?

frp is a fast reverse proxy which can help you expose a local server behind a NAT or firewall to the internet.

## Status

frp is under development and you can try it with available version 0.1.

## Quick Start

Read the [QuickStart](doc/quick_start_en.md) | [使用文档](doc/quick_start_zh.md)

## Architecture

![architecture](doc/pic/architecture.png)

## What can I do with frp?

* Expose any http service behind a NAT or firewall to the internet by a server with public IP address.
* Expose any tcp service behind a NAT or firewall to the internet by a server with public IP address.
* Inspect all http requests/responses that are transmitted over the tunnel(future).
