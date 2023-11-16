### Features

* New command line parameter `--strict_config` is added to enable strict configuration validation mode. It will throw an error for non-existent fields instead of ignoring them.

### Fixes

* frpc: Return code 1 when the first login attempt fails and exits.
* When auth.method is `oidc` and auth.additionalScopes contains `HeartBeats`, if obtaining AccessToken fails, the application will be unresponsive.
