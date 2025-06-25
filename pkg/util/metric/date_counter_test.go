package metric

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDateCounter(t *testing.T) {
	require := require.New(t)

	dc := NewDateCounter(3)
	dc.Inc(10)
	require.EqualValues(10, dc.TodayCount())

	dc.Dec(5)
	require.EqualValues(5, dc.TodayCount())

	counts := dc.GetLastDaysCount(3)
	require.EqualValues(3, len(counts))
	require.EqualValues(5, counts[0])
	require.EqualValues(0, counts[1])
	require.EqualValues(0, counts[2])

	dcTmp := dc.Snapshot()
	require.EqualValues(5, dcTmp.TodayCount())
}
