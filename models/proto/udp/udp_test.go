package udp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUdpPacket(t *testing.T) {
	assert := assert.New(t)

	buf := []byte("hello world")
	udpMsg := NewUDPPacket(buf, nil, nil)

	newBuf, err := GetContent(udpMsg)
	assert.NoError(err)
	assert.EqualValues(buf, newBuf)
}
