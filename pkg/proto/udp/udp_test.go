package udp

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/fatedier/frp/pkg/msg"
)

func TestUdpPacket(t *testing.T) {
	require := require.New(t)

	buf := []byte("hello world")
	udpMsg := NewUDPPacket(buf, nil, nil)

	newBuf, err := GetContent(udpMsg)
	require.NoError(err)
	require.EqualValues(buf, newBuf)
}

func TestForwardUserConnReturnsWhenSendChannelIsClosed(t *testing.T) {
	listener, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	require.NoError(t, err)
	t.Cleanup(func() { _ = listener.Close() })

	readCh := make(chan *msg.UDPPacket)
	sendCh := make(chan *msg.UDPPacket)
	close(sendCh)
	t.Cleanup(func() { close(readCh) })

	done := make(chan struct{})
	go func() {
		ForwardUserConn(listener, readCh, sendCh, 1500)
		close(done)
	}()

	sender, err := net.DialUDP("udp4", nil, listener.LocalAddr().(*net.UDPAddr))
	require.NoError(t, err)
	t.Cleanup(func() { _ = sender.Close() })

	_, err = sender.Write([]byte("trigger"))
	require.NoError(t, err)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("ForwardUserConn did not return after sending to a closed channel")
	}
}
