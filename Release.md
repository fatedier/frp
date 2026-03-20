## Features

* Added a built-in `store` capability for frpc, including persisted store source (`[store] path = "..."`), Store CRUD admin APIs (`/api/store/proxies*`, `/api/store/visitors*`) with runtime reload, and Store management pages in the frpc web dashboard.

## Improvements

* Kept proxy/visitor names as raw config names during completion; moved user-prefix handling to explicit wire-level naming logic.
* Added `noweb` build tag to allow compiling without frontend assets. `make build` now auto-detects missing `web/*/dist` directories and skips embedding, so a fresh clone can build without running `make web` first. The dashboard gracefully returns 404 when assets are not embedded.
* Improved config parsing errors: for `.toml` files, syntax errors now return immediately with parser position details (line/column when available) instead of falling through to YAML/JSON parsing, and TOML type mismatches report field-level errors without misleading line numbers.
* OIDC auth now caches the access token and refreshes it before expiry, avoiding a new token request on every heartbeat. Falls back to per-request fetch when the provider omits `expires_in`.
