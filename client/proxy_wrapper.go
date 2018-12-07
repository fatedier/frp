package client

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fatedier/frp/models/config"
	"github.com/fatedier/frp/models/msg"
	"github.com/fatedier/frp/utils/log"
	frpNet "github.com/fatedier/frp/utils/net"
)

const (
	ProxyStatusNew         = "new"
	ProxyStatusWaitStart   = "wait start"
	ProxyStatusStartErr    = "start error"
	ProxyStatusRunning     = "running"
	ProxyStatusCheckFailed = "check failed"
	ProxyStatusClosed      = "closed"
)

var (
	statusCheckInterval time.Duration = 3 * time.Second
	waitResponseTimeout               = 20 * time.Second
	startErrTimeout                   = 30 * time.Second
)

type ProxyStatus struct {
	Name   string           `json:"name"`
	Type   string           `json:"type"`
	Status string           `json:"status"`
	Err    string           `json:"err"`
	Cfg    config.ProxyConf `json:"cfg"`

	// Got from server.
	RemoteAddr string `json:"remote_addr"`
}

type ProxyWrapper struct {
	ProxyStatus

	// underlying proxy
	pxy Proxy

	// if ProxyConf has healcheck config
	// monitor will watch if it is alive
	monitor *HealthCheckMonitor

	// event handler
	handler EventHandler

	health           uint32
	lastSendStartMsg time.Time
	lastStartErr     time.Time
	closeCh          chan struct{}
	mu               sync.RWMutex

	log.Logger
}

func NewProxyWrapper(cfg config.ProxyConf, eventHandler EventHandler, logPrefix string) *ProxyWrapper {
	baseInfo := cfg.GetBaseInfo()
	pw := &ProxyWrapper{
		ProxyStatus: ProxyStatus{
			Name:   baseInfo.ProxyName,
			Type:   baseInfo.ProxyType,
			Status: ProxyStatusNew,
			Cfg:    cfg,
		},
		closeCh: make(chan struct{}),
		handler: eventHandler,
		Logger:  log.NewPrefixLogger(logPrefix),
	}
	pw.AddLogPrefix(pw.Name)

	if baseInfo.HealthCheckType != "" {
		pw.health = 1 // means failed
		pw.monitor = NewHealthCheckMonitor(baseInfo.HealthCheckType, baseInfo.HealthCheckIntervalS,
			baseInfo.HealthCheckTimeoutS, baseInfo.HealthCheckMaxFailed, baseInfo.HealthCheckAddr,
			baseInfo.HealthCheckUrl, pw.statusNormalCallback, pw.statusFailedCallback)
		pw.monitor.SetLogger(pw.Logger)
		pw.Trace("enable health check monitor")
	}

	pw.pxy = NewProxy(pw.Cfg)
	return pw
}

func (pw *ProxyWrapper) SetRunningStatus(remoteAddr string, respErr string) error {
	pw.mu.Lock()
	defer pw.mu.Unlock()
	if pw.Status != ProxyStatusWaitStart {
		return fmt.Errorf("status not wait start, ignore start message")
	}

	pw.RemoteAddr = remoteAddr
	if respErr != "" {
		pw.Status = ProxyStatusStartErr
		pw.Err = respErr
		pw.lastStartErr = time.Now()
		return fmt.Errorf(pw.Err)
	}

	if err := pw.pxy.Run(); err != nil {
		pw.Status = ProxyStatusStartErr
		pw.Err = err.Error()
		pw.lastStartErr = time.Now()
		return err
	}

	pw.Status = ProxyStatusRunning
	pw.Err = ""
	return nil
}

func (pw *ProxyWrapper) Start() {
	go pw.checkWorker()
	if pw.monitor != nil {
		go pw.monitor.Start()
	}
}

func (pw *ProxyWrapper) Stop() {
	pw.mu.Lock()
	defer pw.mu.Unlock()
	pw.pxy.Close()
	if pw.monitor != nil {
		pw.monitor.Stop()
	}
	pw.Status = ProxyStatusClosed

	pw.handler(EvCloseProxy, &CloseProxyPayload{
		CloseProxyMsg: &msg.CloseProxy{
			ProxyName: pw.Name,
		},
	})
}

func (pw *ProxyWrapper) checkWorker() {
	for {
		// check proxy status
		now := time.Now()
		if atomic.LoadUint32(&pw.health) == 0 {
			pw.mu.Lock()
			if pw.Status == ProxyStatusNew ||
				pw.Status == ProxyStatusCheckFailed ||
				(pw.Status == ProxyStatusWaitStart && now.After(pw.lastSendStartMsg.Add(waitResponseTimeout))) ||
				(pw.Status == ProxyStatusStartErr && now.After(pw.lastStartErr.Add(startErrTimeout))) {

				pw.Trace("change status from [%s] to [%s]", pw.Status, ProxyStatusWaitStart)
				pw.Status = ProxyStatusWaitStart

				var newProxyMsg msg.NewProxy
				pw.Cfg.MarshalToMsg(&newProxyMsg)
				pw.lastSendStartMsg = now
				pw.handler(EvStartProxy, &StartProxyPayload{
					NewProxyMsg: &newProxyMsg,
				})
			}
			pw.mu.Unlock()
		} else {
			pw.mu.Lock()
			if pw.Status == ProxyStatusRunning || pw.Status == ProxyStatusWaitStart {
				pw.handler(EvCloseProxy, &CloseProxyPayload{
					CloseProxyMsg: &msg.CloseProxy{
						ProxyName: pw.Name,
					},
				})
				pw.Trace("change status from [%s] to [%s]", pw.Status, ProxyStatusCheckFailed)
				pw.Status = ProxyStatusCheckFailed
			}
			pw.mu.Unlock()
		}

		select {
		case <-pw.closeCh:
			return
		case <-time.After(statusCheckInterval):
		}
	}
}

func (pw *ProxyWrapper) statusNormalCallback() {
	atomic.StoreUint32(&pw.health, 0)
	pw.Info("health check success")
}

func (pw *ProxyWrapper) statusFailedCallback() {
	atomic.StoreUint32(&pw.health, 1)
	pw.Info("health check failed")
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
