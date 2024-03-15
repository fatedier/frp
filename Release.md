### Features

* Support range ports mapping in TOML/YAML/JSON configuration file by using go template syntax.

  For example:

  ```
  {{- range $_, $v := parseNumberRangePair "6000-6006,6007" "6000-6006,6007" }}
  [[proxies]]
  name = "tcp-{{ $v.First }}"
  type = "tcp"
  localPort = {{ $v.First }}
  remotePort = {{ $v.Second }}
  {{- end }}
  ```

  This will create 8 proxies such as `tcp-6000, tcp-6001, ... tcp-6007`.

### Fixes

* Fix the issue of incorrect interval time for rotating the log by day.
* Disable quic-go's ECN support by default. It may cause issues on certain operating systems.
