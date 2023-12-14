### Features

* The new command line parameter `--strict_config` has been added to enable strict configuration validation mode. It will throw an error for unknown fields instead of ignoring them. In future versions, we will set the default value of this parameter to true to avoid misconfigurations.
* Support `SSH reverse tunneling`. With this feature, you can expose your local service without running frpc, only using SSH. The SSH reverse tunnel agent has many functional limitations compared to the frpc agent. The currently supported proxy types are tcp, http, https, tcpmux, and stcp.
* The frpc tcpmux command line parameters have been updated to support configuring `http_user` and `http_pwd`.
* The frpc stcp/sudp/xtcp command line parameters have been updated to support configuring `allow_users`.

### Fixes

* frpc: Return code 1 when the first login attempt fails and exits.
* When auth.method is `oidc` and auth.additionalScopes contains `HeartBeats`, if obtaining AccessToken fails, the application will be unresponsive.
