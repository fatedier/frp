package util

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRandId(t *testing.T) {
	require := require.New(t)
	id, err := RandID()
	require.NoError(err)
	t.Log(id)
	require.Equal(16, len(id))
}

func TestGetAuthKey(t *testing.T) {
	require := require.New(t)
	key := GetAuthKey("1234", 1488720000)
	require.Equal("6df41a43725f0c770fd56379e12acf8c", key)
}

func TestParseRangeNumbers(t *testing.T) {
	require := require.New(t)
	numbers, err := ParseRangeNumbers("2-5")
	require.NoError(err)
	require.Equal([]int64{2, 3, 4, 5}, numbers)

	numbers, err = ParseRangeNumbers("1")
	require.NoError(err)
	require.Equal([]int64{1}, numbers)

	numbers, err = ParseRangeNumbers("3-5,8")
	require.NoError(err)
	require.Equal([]int64{3, 4, 5, 8}, numbers)

	numbers, err = ParseRangeNumbers(" 3-5,8, 10-12 ")
	require.NoError(err)
	require.Equal([]int64{3, 4, 5, 8, 10, 11, 12}, numbers)

	_, err = ParseRangeNumbers("3-a")
	require.Error(err)
}

func TestCanonicalAddr(t *testing.T) {
	require := require.New(t)

	require.Equal("example.com", CanonicalAddr("example.com", 80))
	require.Equal("example.com", CanonicalAddr("example.com", 443))
	require.Equal("example.com:8080", CanonicalAddr("example.com", 8080))

	// Bare IPv6 must stay bracketed even when the default port is omitted.
	require.Equal("[::1]", CanonicalAddr("::1", 80))
	require.Equal("[2001:db8::1]", CanonicalAddr("2001:db8::1", 443))
	require.Equal("[::1]:8080", CanonicalAddr("::1", 8080))

	// Already-bracketed IPv6 and IPv4 stay unchanged for default ports.
	require.Equal("[::1]", CanonicalAddr("[::1]", 80))
	require.Equal("127.0.0.1", CanonicalAddr("127.0.0.1", 80))
	require.Equal("127.0.0.1:8443", CanonicalAddr("127.0.0.1", 8443))
}

func TestClonePtr(t *testing.T) {
	require := require.New(t)

	var nilInt *int
	require.Nil(ClonePtr(nilInt))

	v := 42
	cloned := ClonePtr(&v)
	require.NotNil(cloned)
	require.Equal(v, *cloned)
	require.NotSame(&v, cloned)
}
