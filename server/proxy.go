package server

import (
	"fmt"
	"io"

	"github.com/fatedier/frp/models/config"
	"github.com/fatedier/frp/models/msg"
	"github.com/fatedier/frp/models/proto/tcp"
	"github.com/fatedier/frp/utils/log"
	"github.com/fatedier/frp/utils/net"
)

type Proxy interface {
	Run() error
	GetControl() *Control
	GetName() string
	GetConf() config.ProxyConf
	Close()
	log.Logger
}

type BaseProxy struct {
	name      string
	ctl       *Control
	listeners []net.Listener
	log.Logger
}

func (pxy *BaseProxy) GetName() string {
	return pxy.name
}

func (pxy *BaseProxy) GetControl() *Control {
	return pxy.ctl
}

func (pxy *BaseProxy) Close() {
	pxy.Info("proxy closing")
	for _, l := range pxy.listeners {
		l.Close()
	}
}

func NewProxy(ctl *Control, pxyConf config.ProxyConf) (pxy Proxy, err error) {
	basePxy := BaseProxy{
		name:      pxyConf.GetName(),
		ctl:       ctl,
		listeners: make([]net.Listener, 0),
		Logger:    log.NewPrefixLogger(ctl.runId),
	}
	switch cfg := pxyConf.(type) {
	case *config.TcpProxyConf:
		pxy = &TcpProxy{
			BaseProxy: basePxy,
			cfg:       cfg,
		}
	case *config.HttpProxyConf:
		pxy = &HttpProxy{
			BaseProxy: basePxy,
			cfg:       cfg,
		}
	case *config.HttpsProxyConf:
		pxy = &HttpsProxy{
			BaseProxy: basePxy,
			cfg:       cfg,
		}
	case *config.UdpProxyConf:
		pxy = &UdpProxy{
			BaseProxy: basePxy,
			cfg:       cfg,
		}
	default:
		return pxy, fmt.Errorf("proxy type not support")
	}
	pxy.AddLogPrefix(pxy.GetName())
	return
}

type TcpProxy struct {
	BaseProxy
	cfg *config.TcpProxyConf
}

func (pxy *TcpProxy) Run() error {
	listener, err := net.ListenTcp(config.ServerCommonCfg.BindAddr, int64(pxy.cfg.RemotePort))
	if err != nil {
		return err
	}
	pxy.listeners = append(pxy.listeners, listener)

	go func(l net.Listener) {
		for {
			// block
			// if listener is closed, err returned
			c, err := l.Accept()
			if err != nil {
				pxy.Info("listener is closed")
				return
			}
			pxy.Debug("got one user connection [%s]", c.RemoteAddr().String())
			go HandleUserTcpConnection(pxy, c)
		}
	}(listener)
	return nil
}

func (pxy *TcpProxy) GetConf() config.ProxyConf {
	return pxy.cfg
}

func (pxy *TcpProxy) Close() {
	pxy.BaseProxy.Close()
}

type HttpProxy struct {
	BaseProxy
	cfg *config.HttpProxyConf
}

func (pxy *HttpProxy) Run() (err error) {
	return
}

func (pxy *HttpProxy) GetConf() config.ProxyConf {
	return pxy.cfg
}

func (pxy *HttpProxy) Close() {
	pxy.BaseProxy.Close()
}

type HttpsProxy struct {
	BaseProxy
	cfg *config.HttpsProxyConf
}

func (pxy *HttpsProxy) Run() (err error) {
	return
}

func (pxy *HttpsProxy) GetConf() config.ProxyConf {
	return pxy.cfg
}

func (pxy *HttpsProxy) Close() {
	pxy.BaseProxy.Close()
}

type UdpProxy struct {
	BaseProxy
	cfg *config.UdpProxyConf
}

func (pxy *UdpProxy) Run() (err error) {
	return
}

func (pxy *UdpProxy) GetConf() config.ProxyConf {
	return pxy.cfg
}

func (pxy *UdpProxy) Close() {
	pxy.BaseProxy.Close()
}

// HandleUserTcpConnection is used for incoming tcp user connections.
// It can be used for tcp, http, https type.
func HandleUserTcpConnection(pxy Proxy, userConn net.Conn) {
	defer userConn.Close()
	ctl := pxy.GetControl()
	var (
		workConn net.Conn
		err      error
	)
	// try all connections from the pool
	for i := 0; i < ctl.poolCount+1; i++ {
		if workConn, err = ctl.GetWorkConn(); err != nil {
			pxy.Warn("failed to get work connection: %v", err)
			return
		}
		defer workConn.Close()
		pxy.Info("get one new work connection: %s", workConn.RemoteAddr().String())
		workConn.AddLogPrefix(pxy.GetName())

		err := msg.WriteMsg(workConn, &msg.StartWorkConn{
			ProxyName: pxy.GetName(),
		})
		if err != nil {
			workConn.Warn("failed to send message to work connection from pool: %v, times: %d", err, i)
			workConn.Close()
		} else {
			break
		}
	}

	if err != nil {
		pxy.Error("try to get work connection failed in the end")
		return
	}

	var (
		local  io.ReadWriteCloser
		remote io.ReadWriteCloser
	)
	local = workConn
	remote = userConn
	cfg := pxy.GetConf().GetBaseInfo()
	if cfg.UseEncryption {
		local, err = tcp.WithEncryption(local, []byte(config.ServerCommonCfg.PrivilegeToken))
		if err != nil {
			pxy.Error("create encryption stream error: %v", err)
			return
		}
	}
	if cfg.UseCompression {
		local = tcp.WithCompression(local)
	}
	tcp.Join(local, remote)
}
