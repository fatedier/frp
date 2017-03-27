package metric

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDateCounter(t *testing.T) {
	assert := assert.New(t)

	dc := NewDateCounter(3)
	dc.Inc(10)
	assert.EqualValues(10, dc.TodayCount())

	dc.Dec(5)
	assert.EqualValues(5, dc.TodayCount())

	counts := dc.GetLastDaysCount(3)
	assert.EqualValues(3, len(counts))
	assert.EqualValues(5, counts[0])
	assert.EqualValues(0, counts[1])
	assert.EqualValues(0, counts[2])

	dcTmp := dc.Snapshot()
	assert.EqualValues(5, dcTmp.TodayCount())
}
