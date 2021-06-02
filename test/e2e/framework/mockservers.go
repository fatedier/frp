package framework

import (
	"fmt"
	"os"

	"github.com/fatedier/frp/test/e2e/mock/server"
	"github.com/fatedier/frp/test/e2e/pkg/port"
)

const (
	TCPEchoServerPort = "TCPEchoServerPort"
	UDPEchoServerPort = "UDPEchoServerPort"
	UDSEchoServerAddr = "UDSEchoServerAddr"
)

type MockServers struct {
	tcpEchoServer *server.Server
	udpEchoServer *server.Server
	udsEchoServer *server.Server
}

func NewMockServers(portAllocator *port.Allocator) *MockServers {
	s := &MockServers{}
	tcpPort := portAllocator.Get()
	udpPort := portAllocator.Get()
	s.tcpEchoServer = server.New(server.TCP, server.WithBindPort(tcpPort), server.WithEchoMode(true))
	s.udpEchoServer = server.New(server.UDP, server.WithBindPort(udpPort), server.WithEchoMode(true))

	udsIndex := portAllocator.Get()
	udsAddr := fmt.Sprintf("%s/frp_echo_server_%d.sock", os.TempDir(), udsIndex)
	os.Remove(udsAddr)
	s.udsEchoServer = server.New(server.Unix, server.WithBindAddr(udsAddr), server.WithEchoMode(true))
	return s
}

func (m *MockServers) Run() error {
	if err := m.tcpEchoServer.Run(); err != nil {
		return err
	}
	if err := m.udpEchoServer.Run(); err != nil {
		return err
	}
	if err := m.udsEchoServer.Run(); err != nil {
		return err
	}
	return nil
}

func (m *MockServers) Close() {
	m.tcpEchoServer.Close()
	m.udpEchoServer.Close()
	m.udsEchoServer.Close()
	os.Remove(m.udsEchoServer.BindAddr())
}

func (m *MockServers) GetTemplateParams() map[string]interface{} {
	ret := make(map[string]interface{})
	ret[TCPEchoServerPort] = m.tcpEchoServer.BindPort()
	ret[UDPEchoServerPort] = m.udpEchoServer.BindPort()
	ret[UDSEchoServerAddr] = m.udsEchoServer.BindAddr()
	return ret
}

func (m *MockServers) GetParam(key string) interface{} {
	params := m.GetTemplateParams()
	if v, ok := params[key]; ok {
		return v
	}
	return nil
}
