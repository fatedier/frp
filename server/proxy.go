package server

import (
	"context"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/fatedier/frp/models/config"
	"github.com/fatedier/frp/models/msg"
	"github.com/fatedier/frp/models/proto/tcp"
	"github.com/fatedier/frp/models/proto/udp"
	"github.com/fatedier/frp/utils/errors"
	"github.com/fatedier/frp/utils/log"
	frpNet "github.com/fatedier/frp/utils/net"
	"github.com/fatedier/frp/utils/vhost"
)

type Proxy interface {
	Run() error
	GetControl() *Control
	GetName() string
	GetConf() config.ProxyConf
	GetWorkConnFromPool() (workConn frpNet.Conn, err error)
	Close()
	log.Logger
}

type BaseProxy struct {
	name      string
	ctl       *Control
	listeners []frpNet.Listener
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

func (pxy *BaseProxy) GetWorkConnFromPool() (workConn frpNet.Conn, err error) {
	ctl := pxy.GetControl()
	// try all connections from the pool
	for i := 0; i < ctl.poolCount+1; i++ {
		if workConn, err = ctl.GetWorkConn(); err != nil {
			pxy.Warn("failed to get work connection: %v", err)
			return
		}
		pxy.Info("get a new work connection: [%s]", workConn.RemoteAddr().String())
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
	return
}

// startListenHandler start a goroutine handler for each listener.
// p: p will just be passed to handler(Proxy, frpNet.Conn).
// handler: each proxy type can set different handler function to deal with connections accepted from listeners.
func (pxy *BaseProxy) startListenHandler(p Proxy, handler func(Proxy, frpNet.Conn)) {
	for _, listener := range pxy.listeners {
		go func(l frpNet.Listener) {
			for {
				// block
				// if listener is closed, err returned
				c, err := l.Accept()
				if err != nil {
					pxy.Info("listener is closed")
					return
				}
				pxy.Debug("get a user connection [%s]", c.RemoteAddr().String())
				go handler(p, c)
			}
		}(listener)
	}
}

func NewProxy(ctl *Control, pxyConf config.ProxyConf) (pxy Proxy, err error) {
	basePxy := BaseProxy{
		name:      pxyConf.GetName(),
		ctl:       ctl,
		listeners: make([]frpNet.Listener, 0),
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
	listener, err := frpNet.ListenTcp(config.ServerCommonCfg.BindAddr, pxy.cfg.RemotePort)
	if err != nil {
		return err
	}
	listener.AddLogPrefix(pxy.name)
	pxy.listeners = append(pxy.listeners, listener)
	pxy.Info("tcp proxy listen port [%d]", pxy.cfg.RemotePort)

	pxy.startListenHandler(pxy, HandleUserTcpConnection)
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
	routeConfig := &vhost.VhostRouteConfig{
		RewriteHost: pxy.cfg.HostHeaderRewrite,
		Username:    pxy.cfg.HttpUser,
		Password:    pxy.cfg.HttpPwd,
	}

	locations := pxy.cfg.Locations
	if len(locations) == 0 {
		locations = []string{""}
	}
	for _, domain := range pxy.cfg.CustomDomains {
		routeConfig.Domain = domain
		for _, location := range locations {
			routeConfig.Location = location
			l, err := pxy.ctl.svr.VhostHttpMuxer.Listen(routeConfig)
			if err != nil {
				return err
			}
			l.AddLogPrefix(pxy.name)
			pxy.Info("http proxy listen for host [%s] location [%s]", routeConfig.Domain, routeConfig.Location)
			pxy.listeners = append(pxy.listeners, l)
		}
	}

	if pxy.cfg.SubDomain != "" {
		routeConfig.Domain = pxy.cfg.SubDomain + "." + config.ServerCommonCfg.SubDomainHost
		for _, location := range locations {
			routeConfig.Location = location
			l, err := pxy.ctl.svr.VhostHttpMuxer.Listen(routeConfig)
			if err != nil {
				return err
			}
			l.AddLogPrefix(pxy.name)
			pxy.Info("http proxy listen for host [%s] location [%s]", routeConfig.Domain, routeConfig.Location)
			pxy.listeners = append(pxy.listeners, l)
		}
	}

	pxy.startListenHandler(pxy, HandleUserTcpConnection)
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
	routeConfig := &vhost.VhostRouteConfig{}

	for _, domain := range pxy.cfg.CustomDomains {
		routeConfig.Domain = domain
		l, err := pxy.ctl.svr.VhostHttpsMuxer.Listen(routeConfig)
		if err != nil {
			return err
		}
		l.AddLogPrefix(pxy.name)
		pxy.Info("https proxy listen for host [%s]", routeConfig.Domain)
		pxy.listeners = append(pxy.listeners, l)
	}

	if pxy.cfg.SubDomain != "" {
		routeConfig.Domain = pxy.cfg.SubDomain + "." + config.ServerCommonCfg.SubDomainHost
		l, err := pxy.ctl.svr.VhostHttpsMuxer.Listen(routeConfig)
		if err != nil {
			return err
		}
		l.AddLogPrefix(pxy.name)
		pxy.Info("https proxy listen for host [%s]", routeConfig.Domain)
		pxy.listeners = append(pxy.listeners, l)
	}

	pxy.startListenHandler(pxy, HandleUserTcpConnection)
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

	udpConn      *net.UDPConn
	workConn     net.Conn
	sendCh       chan *msg.UdpPacket
	readCh       chan *msg.UdpPacket
	checkCloseCh chan int
}

func (pxy *UdpProxy) Run() (err error) {
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", config.ServerCommonCfg.BindAddr, pxy.cfg.RemotePort))
	if err != nil {
		return err
	}
	udpConn, err := net.ListenUDP("udp", addr)
	if err != nil {
		pxy.Warn("listen udp port error: %v", err)
		return err
	}
	pxy.Info("udp proxy listen port [%d]", pxy.cfg.RemotePort)

	pxy.udpConn = udpConn
	pxy.sendCh = make(chan *msg.UdpPacket, 64)
	pxy.readCh = make(chan *msg.UdpPacket, 64)
	pxy.checkCloseCh = make(chan int)

	workConnReaderFn := func(conn net.Conn) {
		for {
			var udpMsg msg.UdpPacket
			if errRet := msg.ReadMsgInto(conn, &udpMsg); errRet != nil {
				pxy.Warn("read from workConn for udp error: %v", errRet)
				conn.Close()
				// notity proxy to start a new work connection
				errors.PanicToError(func() {
					pxy.checkCloseCh <- 1
				})
				return
			}
			if errRet := errors.PanicToError(func() {
				pxy.readCh <- &udpMsg
			}); errRet != nil {
				pxy.Info("reader goroutine for udp work connection closed")
				return
			}
		}
	}
	workConnSenderFn := func(conn net.Conn, ctx context.Context) {
		var errRet error
		for {
			select {
			case udpMsg, ok := <-pxy.sendCh:
				if !ok {
					return
				}
				if errRet = msg.WriteMsg(conn, udpMsg); errRet != nil {
					pxy.Info("sender goroutine for udp work connection closed: %v", errRet)
					return
				} else {
					continue
				}
			case <-ctx.Done():
				pxy.Info("sender goroutine for udp work connection closed")
				return
			}
		}
	}

	go func() {
		for {
			// Sleep a while for waiting control send the NewProxyResp to client.
			time.Sleep(500 * time.Millisecond)
			workConn, err := pxy.GetWorkConnFromPool()
			if err != nil {
				time.Sleep(5 * time.Second)
				// check if proxy is closed
				select {
				case _, ok := <-pxy.checkCloseCh:
					if !ok {
						return
					}
				default:
				}
				continue
			}
			pxy.workConn = workConn
			ctx, cancel := context.WithCancel(context.Background())
			go workConnReaderFn(workConn)
			go workConnSenderFn(workConn, ctx)
			_, ok := <-pxy.checkCloseCh
			cancel()
			if !ok {
				return
			}
		}
	}()

	// Read from user connections and send wrapped udp message to sendCh.
	// Client will transfor udp message to local udp service and waiting for response for a while.
	// Response will be wrapped to be transfored in work connection to server.
	udp.ForwardUserConn(udpConn, pxy.readCh, pxy.sendCh)
	return nil
}

func (pxy *UdpProxy) GetConf() config.ProxyConf {
	return pxy.cfg
}

func (pxy *UdpProxy) Close() {
	pxy.BaseProxy.Close()
	pxy.workConn.Close()
	pxy.udpConn.Close()
	close(pxy.checkCloseCh)
	close(pxy.readCh)
	close(pxy.sendCh)
}

// HandleUserTcpConnection is used for incoming tcp user connections.
// It can be used for tcp, http, https type.
func HandleUserTcpConnection(pxy Proxy, userConn frpNet.Conn) {
	defer userConn.Close()

	// try all connections from the pool
	workConn, err := pxy.GetWorkConnFromPool()
	if err != nil {
		return
	}
	defer workConn.Close()

	var local io.ReadWriteCloser = workConn
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
	pxy.Debug("join connections, workConn(l[%s] r[%s]) userConn(l[%s] r[%s])", workConn.LocalAddr().String(),
		workConn.RemoteAddr().String(), userConn.LocalAddr().String(), userConn.RemoteAddr().String())
	tcp.Join(local, userConn)
	pxy.Debug("join connections closed")
}
