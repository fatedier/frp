## Fixes

* Fixed a server panic and remote denial of service caused by a client sending a negative `pool_count`. Negative values are now rejected before work-connection pool resources are allocated.
