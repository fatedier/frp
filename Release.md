## Notes

We have thoroughly refactored xtcp in this version to improve its penetration rate and stability.

In this version, different penetration strategies can be attempted by retrying connections multiple times. Once a hole is successfully punched, the strategy will be recorded in the server cache for future reuse. When new users connect, the successfully penetrated tunnel can be reused instead of punching a new hole.

**Due to a significant refactor of xtcp, this version is not compatible with previous versions of xtcp.**

**To use features related to xtcp, both frpc and frps need to be updated to the latest version.**

### New

* The frpc has added the `nathole discover` command for testing the NAT type of the current network.
* `XTCP` has been refactored, resulting in a significant improvement in the success rate of penetration.
* When verifying passwords, use `subtle.ConstantTimeCompare` and introduce a certain delay when the password is incorrect.

### Fix

* Fix the problem of lagging when opening multiple table entries in the frps dashboard.
