package metric

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCounter(t *testing.T) {
	require := require.New(t)
	c := NewCounter()
	c.Inc(10)
	require.EqualValues(10, c.Count())

	c.Dec(5)
	require.EqualValues(5, c.Count())

	cTmp := c.Snapshot()
	require.EqualValues(5, cTmp.Count())

	c.Clear()
	require.EqualValues(0, c.Count())
}
