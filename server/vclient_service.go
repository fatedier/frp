package server

import (
	"context"
	"fmt"
	"net"
	"sync/atomic"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/fatedier/frp/pkg/config"
	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/msg"
	plugin "github.com/fatedier/frp/pkg/plugin/server"
	"github.com/fatedier/frp/pkg/util/log"
	frp_net "github.com/fatedier/frp/pkg/util/net"
	"github.com/fatedier/frp/pkg/util/util"
	"github.com/fatedier/frp/pkg/util/xlog"
	"github.com/fatedier/frp/server/controller"
	"github.com/fatedier/frp/server/proxy"
)

// VirtualService is a client VirtualService run in frps
type VirtualService struct {
	clientCfg v1.ClientCommonConfig
	pxyCfg    v1.ProxyConfigurer
	serverCfg v1.ServerConfig

	sshSvc *SSHService

	// uniq id got from frps, attach it in loginMsg
	runID    string
	loginMsg *msg.Login

	// All resource managers and controllers
	rc *controller.ResourceController

	exit uint32 // 0 means not exit
	// SSHService context
	ctx context.Context
	// call cancel to stop SSHService
	cancel context.CancelFunc

	replyCh chan interface{}
	pxy     proxy.Proxy
}

func NewVirtualService(
	ctx context.Context,
	clientCfg v1.ClientCommonConfig,
	serverCfg v1.ServerConfig,
	logMsg msg.Login,
	rc *controller.ResourceController,
	pxyCfg v1.ProxyConfigurer,
	sshSvc *SSHService,
	replyCh chan interface{},
) (svr *VirtualService, err error) {
	svr = &VirtualService{
		clientCfg: clientCfg,
		serverCfg: serverCfg,
		rc:        rc,

		loginMsg: &logMsg,

		sshSvc: sshSvc,
		pxyCfg: pxyCfg,

		ctx:  ctx,
		exit: 0,

		replyCh: replyCh,
	}

	svr.runID, err = util.RandID()
	if err != nil {
		return nil, err
	}

	go svr.loopCheck()

	return
}

func (svr *VirtualService) Run(ctx context.Context) (err error) {
	ctx, cancel := context.WithCancel(ctx)
	svr.ctx = xlog.NewContext(ctx, xlog.New())
	svr.cancel = cancel

	remoteAddr, err := svr.RegisterProxy(&msg.NewProxy{
		ProxyName:  svr.pxyCfg.(*v1.TCPProxyConfig).Name,
		ProxyType:  svr.pxyCfg.(*v1.TCPProxyConfig).Type,
		RemotePort: svr.pxyCfg.(*v1.TCPProxyConfig).RemotePort,
	})
	if err != nil {
		return err
	}

	log.Info("run a reverse proxy on port: %v", remoteAddr)

	return nil
}

func (svr *VirtualService) Close() {
	svr.GracefulClose(time.Duration(0))
}

func (svr *VirtualService) GracefulClose(d time.Duration) {
	atomic.StoreUint32(&svr.exit, 1)
	svr.pxy.Close()

	if svr.cancel != nil {
		svr.cancel()
	}

	svr.replyCh <- &VProxyError{}
}

func (svr *VirtualService) loopCheck() {
	<-svr.sshSvc.Exit()
	svr.pxy.Close()
	log.Info("virtual client service close")
}

func (svr *VirtualService) RegisterProxy(pxyMsg *msg.NewProxy) (remoteAddr string, err error) {
	var pxyConf v1.ProxyConfigurer
	pxyConf, err = config.NewProxyConfigurerFromMsg(pxyMsg, &svr.serverCfg)
	if err != nil {
		return
	}

	// User info
	userInfo := plugin.UserInfo{
		User:  svr.loginMsg.User,
		Metas: svr.loginMsg.Metas,
		RunID: svr.runID,
	}

	svr.pxy, err = proxy.NewProxy(svr.ctx, &proxy.Options{
		LoginMsg:           svr.loginMsg,
		UserInfo:           userInfo,
		Configurer:         pxyConf,
		ResourceController: svr.rc,

		GetWorkConnFn: svr.GetWorkConn,
		PoolCount:     10,

		ServerCfg: &svr.serverCfg,
	})
	if err != nil {
		return remoteAddr, err
	}

	remoteAddr, err = svr.pxy.Run()
	if err != nil {
		log.Warn("proxy run error: %v", err)
		return
	}

	defer func() {
		if err != nil {
			log.Warn("proxy close")
			svr.pxy.Close()
		}
	}()

	return
}

func (svr *VirtualService) GetWorkConn() (workConn net.Conn, err error) {
	// tell ssh client open a new stream for work
	payload := forwardedTCPPayload{
		Addr: svr.serverCfg.BindAddr, // TODO refine
		Port: uint32(svr.pxyCfg.(*v1.TCPProxyConfig).RemotePort),
	}

	log.Info("get work conn payload: %v", payload)

	channel, reqs, err := svr.sshSvc.SSHConn().OpenChannel(ChannelTypeServerOpenChannel, ssh.Marshal(payload))
	if err != nil {
		return nil, fmt.Errorf("open ssh channel error: %v", err)
	}
	go ssh.DiscardRequests(reqs)

	workConn = frp_net.WrapReadWriteCloserToConn(channel, svr.sshSvc.tcpConn)
	return workConn, nil
}
