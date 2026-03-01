## Features

* Added a built-in `store` capability for frpc, including persisted store source (`[store] path = "..."`), Store CRUD admin APIs (`/api/store/proxies*`, `/api/store/visitors*`) with runtime reload, and Store management pages in the frpc web dashboard.

## Improvements

* Kept proxy/visitor names as raw config names during completion; moved user-prefix handling to explicit wire-level naming logic.
