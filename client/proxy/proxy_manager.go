package proxy

import (
	"fmt"
	"sync"

	"github.com/fatedier/frp/client/event"
	"github.com/fatedier/frp/models/config"
	"github.com/fatedier/frp/models/msg"
	"github.com/fatedier/frp/utils/log"
	frpNet "github.com/fatedier/frp/utils/net"

	"github.com/fatedier/golib/errors"
)

type ProxyManager struct {
	sendCh  chan (msg.Message)
	proxies map[string]*ProxyWrapper

	closed bool
	mu     sync.RWMutex

	logPrefix string
	log.Logger
}

func NewProxyManager(msgSendCh chan (msg.Message), logPrefix string) *ProxyManager {
	return &ProxyManager{
		proxies:   make(map[string]*ProxyWrapper),
		sendCh:    msgSendCh,
		closed:    false,
		logPrefix: logPrefix,
		Logger:    log.NewPrefixLogger(logPrefix),
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
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	for _, pxy := range pm.proxies {
		pxy.Stop()
	}
}

func (pm *ProxyManager) HandleWorkConn(name string, workConn frpNet.Conn) {
	pm.mu.RLock()
	pw, ok := pm.proxies[name]
	pm.mu.RUnlock()
	if ok {
		pw.InWorkConn(workConn)
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
		pm.Info("proxy removed: %v", delPxyNames)
	}

	addPxyNames := make([]string, 0)
	for name, cfg := range pxyCfgs {
		if _, ok := pm.proxies[name]; !ok {
			pxy := NewProxyWrapper(cfg, pm.HandleEvent, pm.logPrefix)
			pm.proxies[name] = pxy
			addPxyNames = append(addPxyNames, name)

			pxy.Start()
		}
	}
	if len(addPxyNames) > 0 {
		pm.Info("proxy added: %v", addPxyNames)
	}
}
