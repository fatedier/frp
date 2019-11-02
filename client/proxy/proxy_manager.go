package proxy

import (
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/fatedier/frp/client/event"
	"github.com/fatedier/frp/models/config"
	"github.com/fatedier/frp/models/msg"
	"github.com/fatedier/frp/utils/xlog"

	"github.com/fatedier/golib/errors"
)

type ProxyManager struct {
	sendCh  chan (msg.Message)
	proxies map[string]*ProxyWrapper

	closed bool
	mu     sync.RWMutex

	clientCfg config.ClientCommonConf

	// The UDP port that the server is listening on
	serverUDPPort int

	ctx context.Context
}

func NewProxyManager(ctx context.Context, msgSendCh chan (msg.Message), clientCfg config.ClientCommonConf, serverUDPPort int) *ProxyManager {
	return &ProxyManager{
		sendCh:        msgSendCh,
		proxies:       make(map[string]*ProxyWrapper),
		closed:        false,
		clientCfg:     clientCfg,
		serverUDPPort: serverUDPPort,
		ctx:           ctx,
	}
}

func (pm *ProxyManager) StartProxy(name string, remoteAddr string, serverRespErr string) error {
	pm.mu.RLock()
	pxy, ok := pm.proxies[name]
	pm.mu.RUnlock()
	if !ok {
		return fmt.Errorf("proxy [%s] not found", name)
	}

	err := pxy.SetRunningStatus(remoteAddr, serverRespErr)
	if err != nil {
		return err
	}
	return nil
}

func (pm *ProxyManager) Close() {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	for _, pxy := range pm.proxies {
		pxy.Stop()
	}
	pm.proxies = make(map[string]*ProxyWrapper)
}

func (pm *ProxyManager) HandleWorkConn(name string, workConn net.Conn, m *msg.StartWorkConn) {
	pm.mu.RLock()
	pw, ok := pm.proxies[name]
	pm.mu.RUnlock()
	if ok {
		pw.InWorkConn(workConn, m)
	} else {
		workConn.Close()
	}
}

func (pm *ProxyManager) HandleEvent(evType event.EventType, payload interface{}) error {
	var m msg.Message
	switch e := payload.(type) {
	case *event.StartProxyPayload:
		m = e.NewProxyMsg
	case *event.CloseProxyPayload:
		m = e.CloseProxyMsg
	default:
		return event.ErrPayloadType
	}

	err := errors.PanicToError(func() {
		pm.sendCh <- m
	})
	return err
}

func (pm *ProxyManager) GetAllProxyStatus() []*ProxyStatus {
	ps := make([]*ProxyStatus, 0)
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	for _, pxy := range pm.proxies {
		ps = append(ps, pxy.GetStatus())
	}
	return ps
}

func (pm *ProxyManager) Reload(pxyCfgs map[string]config.ProxyConf) {
	xl := xlog.FromContextSafe(pm.ctx)
	pm.mu.Lock()
	defer pm.mu.Unlock()

	delPxyNames := make([]string, 0)
	for name, pxy := range pm.proxies {
		del := false
		cfg, ok := pxyCfgs[name]
		if !ok {
			del = true
		} else {
			if !pxy.Cfg.Compare(cfg) {
				del = true
			}
		}

		if del {
			delPxyNames = append(delPxyNames, name)
			delete(pm.proxies, name)

			pxy.Stop()
		}
	}
	if len(delPxyNames) > 0 {
		xl.Info("proxy removed: %v", delPxyNames)
	}

	addPxyNames := make([]string, 0)
	for name, cfg := range pxyCfgs {
		if _, ok := pm.proxies[name]; !ok {
			pxy := NewProxyWrapper(pm.ctx, cfg, pm.clientCfg, pm.HandleEvent, pm.serverUDPPort)
			pm.proxies[name] = pxy
			addPxyNames = append(addPxyNames, name)

			pxy.Start()
		}
	}
	if len(addPxyNames) > 0 {
		xl.Info("proxy added: %v", addPxyNames)
	}
}
