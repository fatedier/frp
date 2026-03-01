package naming

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAddUserPrefix(t *testing.T) {
	require := require.New(t)
	require.Equal("test", AddUserPrefix("", "test"))
	require.Equal("alice.test", AddUserPrefix("alice", "test"))
}

func TestStripUserPrefix(t *testing.T) {
	require := require.New(t)
	require.Equal("test", StripUserPrefix("", "test"))
	require.Equal("test", StripUserPrefix("alice", "alice.test"))
	require.Equal("alice.test", StripUserPrefix("alice", "alice.alice.test"))
	require.Equal("bob.test", StripUserPrefix("alice", "bob.test"))
}

func TestBuildTargetServerProxyName(t *testing.T) {
	require := require.New(t)
	require.Equal("alice.test", BuildTargetServerProxyName("alice", "", "test"))
	require.Equal("bob.test", BuildTargetServerProxyName("alice", "bob", "test"))
}
