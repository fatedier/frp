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
