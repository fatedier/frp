package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRandId(t *testing.T) {
	assert := assert.New(t)
	id, err := RandId()
	assert.NoError(err)
	t.Log(id)
	assert.Equal(16, len(id))
}

func TestGetAuthKey(t *testing.T) {
	assert := assert.New(t)
	key := GetAuthKey("1234", 1488720000)
	t.Log(key)
	assert.Equal("6df41a43725f0c770fd56379e12acf8c", key)
}

func TestGetPortRanges(t *testing.T) {
	assert := assert.New(t)

	rangesStr := "2000-3000,3001,4000-50000"
	expect := [][2]int64{
		[2]int64{2000, 3000},
		[2]int64{3001, 3001},
		[2]int64{4000, 50000},
	}
	actual, err := GetPortRanges(rangesStr)
	assert.Nil(err)
	t.Log(actual)
	assert.Equal(expect, actual)
}

func TestContainsPort(t *testing.T) {
	assert := assert.New(t)

	rangesStr := "2000-3000,3001,4000-50000"
	portRanges, err := GetPortRanges(rangesStr)
	assert.Nil(err)

	type Case struct {
		Port   int64
		Answer bool
	}
	cases := []Case{
		Case{
			Port:   3001,
			Answer: true,
		},
		Case{
			Port:   3002,
			Answer: false,
		},
		Case{
			Port:   44444,
			Answer: true,
		},
	}
	for _, elem := range cases {
		ok := ContainsPort(portRanges, elem.Port)
		assert.Equal(elem.Answer, ok)
	}
}

func TestPortRangesCut(t *testing.T) {
	assert := assert.New(t)

	rangesStr := "2000-3000,3001,4000-50000"
	portRanges, err := GetPortRanges(rangesStr)
	assert.Nil(err)

	expect := [][2]int64{
		[2]int64{2000, 3000},
		[2]int64{3001, 3001},
		[2]int64{4000, 44443},
		[2]int64{44445, 50000},
	}
	actual := PortRangesCut(portRanges, 44444)
	t.Log(actual)
	assert.Equal(expect, actual)
}
