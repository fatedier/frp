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

func TestParseRangeNumbers(t *testing.T) {
	assert := assert.New(t)
	numbers, err := ParseRangeNumbers("2-5")
	if assert.NoError(err) {
		assert.Equal([]int64{2, 3, 4, 5}, numbers)
	}

	numbers, err = ParseRangeNumbers("1")
	if assert.NoError(err) {
		assert.Equal([]int64{1}, numbers)
	}

	numbers, err = ParseRangeNumbers("3-5,8")
	if assert.NoError(err) {
		assert.Equal([]int64{3, 4, 5, 8}, numbers)
	}

	numbers, err = ParseRangeNumbers(" 3-5,8, 10-12 ")
	if assert.NoError(err) {
		assert.Equal([]int64{3, 4, 5, 8, 10, 11, 12}, numbers)
	}

	_, err = ParseRangeNumbers("3-a")
	assert.Error(err)
}
