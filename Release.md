## Fixes

* Fixed a configuration-dependent authentication bypass in `type = "http"` proxies when `routeByHTTPUser` is used together with `httpUser` / `httpPassword`. This affected proxy-style requests. Proxy-style authentication failures now return `407 Proxy Authentication Required`.
