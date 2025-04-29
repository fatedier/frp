# Install systemd unit

## install binary

```sudo install ./frps /usr/local/bin```

## Create systemd unit

```sudo vi /lib/systemd/syste.frps.service```



```[Unit]
Description=FRP Server Service
After=network.target 

[Service]
Type=simple
ExecStart=/usr/local/bin/frps -c /usr/local/etc/frp/frps.toml
Restart=on-failure
RestartSec=15s

[Install]
WantedBy=multi-user.target

## Enable service

```sudo systemctl daemon-reload
sudo systemctl enable frps.service
sudo systemctl start frps.service
sudo systemctl status frps.service```

