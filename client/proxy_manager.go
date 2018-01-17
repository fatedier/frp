package client

import (
	"fmt"
	"sync"

	"github.com/fatedier/frp/models/config"
	"github.com/fatedier/frp/models/msg"
	"github.com/fatedier/frp/utils/errors"
	"github.com/fatedier/frp/utils/log"
	frpNet "github.com/fatedier/frp/utils/net"
)

const (
	ProxyStatusNew      = "new"
	ProxyStatusStartErr = "start error"
	ProxyStatusRunning  = "running"
	ProxyStatusClosed   = "closed"
)

type ProxyManager struct {
	ctl *Control

	proxies map[string]*ProxyWrapper

	visitorCfgs map[string]config.ProxyConf
	visitors    map[string]Visitor

	sendCh chan (msg.Message)

	closed bool
	mu     sync.RWMutex

	log.Logger
}

type ProxyWrapper struct {
	Name   string
	Type   string
	Status string
	Err    string
	Cfg    config.ProxyConf

	RemoteAddr string

	pxy Proxy

	mu sync.RWMutex
}

type ProxyStatus struct {
	Name   string           `json:"name"`
	Type   string           `json:"type"`
	Status string           `json:"status"`
	Err    string           `json:"err"`
	Cfg    config.ProxyConf `json:"cfg"`

	// Got from server.
	RemoteAddr string `json:"remote_addr"`
}

func NewProxyWrapper(cfg config.ProxyConf) *ProxyWrapper {
	return &ProxyWrapper{
		Name:   cfg.GetName(),
		Type:   cfg.GetType(),
		Status: ProxyStatusNew,
		Cfg:    cfg,
		pxy:    nil,
	}
}

func (pw *ProxyWrapper) IsRunning() bool {
	pw.mu.RLock()
	defer pw.mu.RUnlock()
	if pw.Status == ProxyStatusRunning {
		return true
	} else {
		return false
	}
}

func (pw *ProxyWrapper) GetStatus() *ProxyStatus {
	pw.mu.RLock()
	defer pw.mu.RUnlock()
	ps := &ProxyStatus{
		Name:       pw.Name,
		Type:       pw.Type,
		Status:     pw.Status,
		Err:        pw.Err,
		Cfg:        pw.Cfg,
		RemoteAddr: pw.RemoteAddr,
	}
	return ps
}

func (pw *ProxyWrapper) Start(remoteAddr string, serverRespErr string) error {
	if pw.pxy != nil {
		pw.pxy.Close()
		pw.pxy = nil
	}

	if serverRespErr != "" {
		pw.mu.Lock()
		pw.Status = ProxyStatusStartErr
		pw.RemoteAddr = remoteAddr
		pw.Err = serverRespErr
		pw.mu.Unlock()
		return fmt.Errorf(serverRespErr)
	}

	pxy := NewProxy(pw.Cfg)
	pw.mu.Lock()
	defer pw.mu.Unlock()
	pw.RemoteAddr = remoteAddr
	if err := pxy.Run(); err != nil {
		pw.Status = ProxyStatusStartErr
		pw.Err = err.Error()
		return err
	}
	pw.Status = ProxyStatusRunning
	pw.Err = ""
	pw.pxy = pxy
	return nil
}

func (pw *ProxyWrapper) InWorkConn(workConn frpNet.Conn) {
	pw.mu.RLock()
	pxy := pw.pxy
	pw.mu.RUnlock()
	if pxy != nil {
		workConn.Debug("start a new work connection, localAddr: %s remoteAddr: %s", workConn.LocalAddr().String(), workConn.RemoteAddr().String())
		go pxy.InWorkConn(workConn)
	} else {
		workConn.Close()
	}
}

func (pw *ProxyWrapper) Close() {
	pw.mu.Lock()
	defer pw.mu.Unlock()
	if pw.pxy != nil {
		pw.pxy.Close()
		pw.pxy = nil
	}
	pw.Status = ProxyStatusClosed
}

func NewProxyManager(ctl *Control, msgSendCh chan (msg.Message), logPrefix string) *ProxyManager {
	return &ProxyManager{
		ctl:         ctl,
		proxies:     make(map[string]*ProxyWrapper),
		visitorCfgs: make(map[string]config.ProxyConf),
		visitors:    make(map[string]Visitor),
		sendCh:      msgSendCh,
		closed:      false,
		Logger:      log.NewPrefixLogger(logPrefix),
	}
}

func (pm *ProxyManager) Reset(msgSendCh chan (msg.Message), logPrefix string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.closed = false
	pm.sendCh = msgSendCh
	pm.ClearLogPrefix()
	pm.AddLogPrefix(logPrefix)
}

// Must hold the lock before calling this function.
func (pm *ProxyManager) sendMsg(m msg.Message) error {
	err := errors.PanicToError(func() {
		pm.sendCh <- m
	})
	if err != nil {
		pm.closed = true
	}
	return err
}

func (pm *ProxyManager) StartProxy(name string, remoteAddr string, serverRespErr string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	if pm.closed {
		return fmt.Errorf("ProxyManager is closed now")
	}

	pxy, ok := pm.proxies[name]
	if !ok {
		return fmt.Errorf("no proxy found")
	}

	if err := pxy.Start(remoteAddr, serverRespErr); err != nil {
		errRet := err
		err = pm.sendMsg(&msg.CloseProxy{
			ProxyName: name,
		})
		if err != nil {
			errRet = fmt.Errorf("send CloseProxy message error")
		}
		return errRet
	}
	return nil
}

func (pm *ProxyManager) CloseProxies() {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	for _, pxy := range pm.proxies {
		pxy.Close()
	}
}

func (pm *ProxyManager) CheckAndStartProxy() {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	if pm.closed {
		pm.Warn("CheckAndStartProxy error: ProxyManager is closed now")
		return
	}

	for _, pxy := range pm.proxies {
		if !pxy.IsRunning() {
			var newProxyMsg msg.NewProxy
			pxy.Cfg.UnMarshalToMsg(&newProxyMsg)
			err := pm.sendMsg(&newProxyMsg)
			if err != nil {
				pm.Warn("[%s] proxy send NewProxy message error")
				return
			}
		}
	}

	for _, cfg := range pm.visitorCfgs {
		if _, exist := pm.visitors[cfg.GetName()]; !exist {
			pm.Info("try to start visitor [%s]", cfg.GetName())
			visitor := NewVisitor(pm.ctl, cfg)
			err := visitor.Run()
			if err != nil {
				visitor.Warn("start error: %v", err)
				continue
			}
			pm.visitors[cfg.GetName()] = visitor
			visitor.Info("start visitor success")
		}
	}
}

func (pm *ProxyManager) Reload(pxyCfgs map[string]config.ProxyConf, visitorCfgs map[string]config.ProxyConf) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	if pm.closed {
		err := fmt.Errorf("Reload error: ProxyManager is closed now")
		pm.Warn(err.Error())
		return err
	}

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

			pxy.Close()
			err := pm.sendMsg(&msg.CloseProxy{
				ProxyName: name,
			})
			if err != nil {
				err = fmt.Errorf("Reload error: ProxyManager is closed now")
				pm.Warn(err.Error())
				return err
			}
		}
	}
	pm.Info("proxy removed: %v", delPxyNames)

	addPxyNames := make([]string, 0)
	for name, cfg := range pxyCfgs {
		if _, ok := pm.proxies[name]; !ok {
			pxy := NewProxyWrapper(cfg)
			pm.proxies[name] = pxy
			addPxyNames = append(addPxyNames, name)
		}
	}
	pm.Info("proxy added: %v", addPxyNames)

	delVisitorName := make([]string, 0)
	for name, oldVisitorCfg := range pm.visitorCfgs {
		del := false
		cfg, ok := visitorCfgs[name]
		if !ok {
			del = true
		} else {
			if !oldVisitorCfg.Compare(cfg) {
				del = true
			}
		}

		if del {
			delVisitorName = append(delVisitorName, name)
			delete(pm.visitorCfgs, name)
			if visitor, ok := pm.visitors[name]; ok {
				visitor.Close()
			}
			delete(pm.visitors, name)
		}
	}
	pm.Info("visitor removed: %v", delVisitorName)

	addVisitorName := make([]string, 0)
	for name, visitorCfg := range visitorCfgs {
		if _, ok := pm.visitorCfgs[name]; !ok {
			pm.visitorCfgs[name] = visitorCfg
			addVisitorName = append(addVisitorName, name)
		}
	}
	pm.Info("visitor added: %v", addVisitorName)
	return nil
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

func (pm *ProxyManager) GetAllProxyStatus() []*ProxyStatus {
	ps := make([]*ProxyStatus, 0)
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	for _, pxy := range pm.proxies {
		ps = append(ps, pxy.GetStatus())
	}
	return ps
}
