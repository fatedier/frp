package metric

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCounter(t *testing.T) {
	assert := assert.New(t)
	c := NewCounter()
	c.Inc(10)
	assert.EqualValues(10, c.Count())

	c.Dec(5)
	assert.EqualValues(5, c.Count())

	cTmp := c.Snapshot()
	assert.EqualValues(5, cTmp.Count())

	c.Clear()
	assert.EqualValues(0, c.Count())
}
