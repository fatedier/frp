### Features

* Configuration: We now support TOML, YAML, and JSON for configuration. Please note that INI is deprecated and will be removed in future releases. New features will only be available in TOML, YAML, or JSON. Users wanting these new features should switch their configuration format accordingly. #2521

### Breaking Changes

* Change the way to start the visitor through the command line from `frpc stcp --role=visitor xxx` to `frpc stcp visitor xxx`.
* Modified the semantics of the `server_addr` in the command line, no longer including the port. Added the `server_port` parameter to configure the port.
* No longer support range ports mapping in TOML/YAML/JSON.
