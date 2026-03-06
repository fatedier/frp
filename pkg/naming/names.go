package naming

import "strings"

// AddUserPrefix builds the wire-level proxy name for frps by prefixing user.
func AddUserPrefix(user, name string) string {
	if user == "" {
		return name
	}
	return user + "." + name
}

// StripUserPrefix converts a wire-level proxy name to an internal raw name.
// It strips only one exact "{user}." prefix.
func StripUserPrefix(user, name string) string {
	if user == "" {
		return name
	}
	if trimmed, ok := strings.CutPrefix(name, user+"."); ok {
		return trimmed
	}
	return name
}

// BuildTargetServerProxyName resolves visitor target proxy name for wire-level
// protocol messages. serverUser overrides local user when set.
func BuildTargetServerProxyName(localUser, serverUser, serverName string) string {
	if serverUser != "" {
		return AddUserPrefix(serverUser, serverName)
	}
	return AddUserPrefix(localUser, serverName)
}
