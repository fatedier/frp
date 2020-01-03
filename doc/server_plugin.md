### Manage Plugin

frp manage plugin is aim to extend frp's ability without modifing self code.

It runs as a process and listen on a port to provide RPC interface. Before frps doing some operations, frps will send RPC requests to manage plugin and do operations by it's response.

### RPC request

Support HTTP first.

When manage plugin accept the operation request, it can give three different responses.

* Reject operation and return the reason.
* Allow operation and keep original content.
* Allow operation and return modified content.

### Interface

HTTP path can be configured for each manage plugin in frps. Assume here is `/handler`.

Request

```
POST /handler
{
    "version": "0.1.0",
    "op": "Login",
    "content": {
        ... // Operation info
    }
}

Request Header
X-Frp-Reqid: for tracing
```

Response

Error if not return 200 http code.

Reject opeartion

```
{
    "reject": true,
    "reject_reason": "invalid user"
}
```

Allow operation and keep original content

```
{
    "reject": false,
    "unchange": true
}
```

Allow opeartion and modify content

```
{
    "unchange": "false",
    "content": {
        ... // Replaced content
    }
}
```

### Operation

Now it supports `Login` and `NewProxy`.

#### Login

Client login operation

```
{
    "content": {
        "version": <string>,
        "hostname": <string>,
        "os": <string>,
        "arch": <string>,
        "user": <string>,
        "timestamp": <int64>,
        "privilege_key": <string>,
        "run_id": <string>,
        "pool_count": <int>,
        "metas": map<string>string
    }
}
```

#### NewProxy

Create new proxy

```
{
    "content": {
        "user": {
            "user": <string>,
            "metas": map<string>string
        },
        "proxy_name": <string>,
        "proxy_type": <string>,
        "use_encryption": <bool>,
        "use_compression": <bool>,
        "group": <string>,
        "group_key": <string>,

        // tcp and udp only
        "remote_port": <int>,

        // http and https only
        "custom_domains": []<string>,
        "subdomain": <string>,
        "locations": <string>,
        "http_user": <string>,
        "http_pwd": <string>,
        "host_header_rewrite": <string>,
        "headers": map<string>string,

        "metas": map<string>string
    }
}
```

### manage plugin configure

```ini
[common]
bind_port = 7000

[plugin.user-manager]
addr = 127.0.0.1:9000
path = /handler
ops = Login

[plugin.port-manager]
addr = 127.0.0.1:9001
path = /handler
ops = NewProxy
```

addr: plugin listen on.
path: http request url path.
ops: opeartions plugin needs handle.

### meta data

Meta data will be sent to manage plugin in each RCP request.

Meta data start with `meta_`. It can be configured in `common` and each proxy.

```
# frpc.ini
[common]
server_addr = 127.0.0.1
server_port = 7000
user = fake
meta_token = fake
meta_version = 1.0.0

[ssh]
type = tcp
local_port = 22
remote_port = 6000
meta_id = 123
```
