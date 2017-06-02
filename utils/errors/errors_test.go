package errors

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPanicToError(t *testing.T) {
	assert := assert.New(t)

	err := PanicToError(func() {
		panic("test error")
	})
	assert.Contains(err.Error(), "test error")
}
