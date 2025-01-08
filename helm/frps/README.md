# frps Helm Chart

## Install the chart

```console
helm install frps .
```


## Uninstall the chart

```console
helm delete frps
```

## Examples

1. Use `LoadBalancer` service to expose frps.
    ```yaml
    # values.yaml
    service:
      type: LoadBalancer
      settings:
        loadBalancerIP: 34.24.52.92  # Replace with correct IP

    # frpc.yaml
    serverAddr: 34.24.52.92
    serverPort: 7000
    auth:
      method: token
      token: "123456789"
    ```

1. Enable admin web server UI using nginx ingress controller.
    ```yaml
    # values.yaml
    service:
      extras:
        admin:
          enabled: true

    ingress:
      extras:
        admin:
          enabled: true
          className: nginx
          hosts:
            - host: frps-admin.mydomain.com
              paths:
                - path: /
                  pathType: ImplementationSpecific
    ```

1. Enable `vhostHTTPPort` proxying using nginx ingress controller.
    ```yaml
    # values.yaml
    service:
      extras:
        http:
          enabled: true

    ingress:
      extras:
        app1:
          enabled: true
          className: nginx
          hosts:
            - host: app1.mydomain.com
              paths:
                - path: /
                  pathType: ImplementationSpecific
          backendService:
            name: frps-http
            port: 80
        app2:
          enabled: true
          className: nginx
          hosts:
            - host: app2.mydomain.com
              paths:
                - path: /
                  pathType: ImplementationSpecific
          backendService:
            name: frps-http
            port: 80

    # frpc.yaml
    serverAddr: 34.24.52.92
    serverPort: 7000
    auth:
      method: token
      token: "123456789"

    proxies:
    - name: app1
      type: http
      localIP: 127.0.0.1
      localPort: 8081
      customDomains:
      - app1.mydomain.com
    - name: app2
      type: http
      localIP: 127.0.0.1
      localPort: 8082
      customDomains:
      - app2.mydomain.com
    ```

Refer to [values.yaml](./values.yaml) for additional settings.

