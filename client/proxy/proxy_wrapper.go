package proxy

import (
	"context"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fatedier/frp/client/event"
	"github.com/fatedier/frp/client/health"
	"github.com/fatedier/frp/models/config"
	"github.com/fatedier/frp/models/msg"
	"github.com/fatedier/frp/utils/xlog"

	"github.com/fatedier/golib/errors"
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
	monitor *health.HealthCheckMonitor

	// event handler
	handler event.EventHandler

	health           uint32
	lastSendStartMsg time.Time
	lastStartErr     time.Time
	closeCh          chan struct{}
	healthNotifyCh   chan struct{}
	mu               sync.RWMutex

	xl  *xlog.Logger
	ctx context.Context
}

func NewProxyWrapper(ctx context.Context, cfg config.ProxyConf, clientCfg config.ClientCommonConf, eventHandler event.EventHandler, serverUDPPort int) *ProxyWrapper {
	baseInfo := cfg.GetBaseInfo()
	xl := xlog.FromContextSafe(ctx).Spawn().AppendPrefix(baseInfo.ProxyName)
	pw := &ProxyWrapper{
		ProxyStatus: ProxyStatus{
			Name:   baseInfo.ProxyName,
			Type:   baseInfo.ProxyType,
			Status: ProxyStatusNew,
			Cfg:    cfg,
		},
		closeCh:        make(chan struct{}),
		healthNotifyCh: make(chan struct{}),
		handler:        eventHandler,
		xl:             xl,
		ctx:            xlog.NewContext(ctx, xl),
	}

	if baseInfo.HealthCheckType != "" {
		pw.health = 1 // means failed
		pw.monitor = health.NewHealthCheckMonitor(pw.ctx, baseInfo.HealthCheckType, baseInfo.HealthCheckIntervalS,
			baseInfo.HealthCheckTimeoutS, baseInfo.HealthCheckMaxFailed, baseInfo.HealthCheckAddr,
			baseInfo.HealthCheckUrl, pw.statusNormalCallback, pw.statusFailedCallback)
		xl.Trace("enable health check monitor")
	}

	pw.pxy = NewProxy(pw.ctx, pw.Cfg, clientCfg, serverUDPPort)
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
		pw.close()
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
	close(pw.closeCh)
	close(pw.healthNotifyCh)
	pw.pxy.Close()
	if pw.monitor != nil {
		pw.monitor.Stop()
	}
	pw.Status = ProxyStatusClosed
	pw.close()
}

func (pw *ProxyWrapper) close() {
	pw.handler(event.EvCloseProxy, &event.CloseProxyPayload{
		CloseProxyMsg: &msg.CloseProxy{
			ProxyName: pw.Name,
		},
	})
}

func (pw *ProxyWrapper) checkWorker() {
	xl := pw.xl
	if pw.monitor != nil {
		// let monitor do check request first
		time.Sleep(500 * time.Millisecond)
	}
	for {
		// check proxy status
		now := time.Now()
		if atomic.LoadUint32(&pw.health) == 0 {
			pw.mu.Lock()
			if pw.Status == ProxyStatusNew ||
				pw.Status == ProxyStatusCheckFailed ||
				(pw.Status == ProxyStatusWaitStart && now.After(pw.lastSendStartMsg.Add(waitResponseTimeout))) ||
				(pw.Status == ProxyStatusStartErr && now.After(pw.lastStartErr.Add(startErrTimeout))) {

				xl.Trace("change status from [%s] to [%s]", pw.Status, ProxyStatusWaitStart)
				pw.Status = ProxyStatusWaitStart

				var newProxyMsg msg.NewProxy
				pw.Cfg.MarshalToMsg(&newProxyMsg)
				pw.lastSendStartMsg = now
				pw.handler(event.EvStartProxy, &event.StartProxyPayload{
					NewProxyMsg: &newProxyMsg,
				})
			}
			pw.mu.Unlock()
		} else {
			pw.mu.Lock()
			if pw.Status == ProxyStatusRunning || pw.Status == ProxyStatusWaitStart {
				pw.close()
				xl.Trace("change status from [%s] to [%s]", pw.Status, ProxyStatusCheckFailed)
				pw.Status = ProxyStatusCheckFailed
			}
			pw.mu.Unlock()
		}

		select {
		case <-pw.closeCh:
			return
		case <-time.After(statusCheckInterval):
		case <-pw.healthNotifyCh:
		}
	}
}

func (pw *ProxyWrapper) statusNormalCallback() {
	xl := pw.xl
	atomic.StoreUint32(&pw.health, 0)
	errors.PanicToError(func() {
		select {
		case pw.healthNotifyCh <- struct{}{}:
		default:
		}
	})
	xl.Info("health check success")
}

func (pw *ProxyWrapper) statusFailedCallback() {
	xl := pw.xl
	atomic.StoreUint32(&pw.health, 1)
	errors.PanicToError(func() {
		select {
		case pw.healthNotifyCh <- struct{}{}:
		default:
		}
	})
	xl.Info("health check failed")
}

func (pw *ProxyWrapper) InWorkConn(workConn net.Conn, m *msg.StartWorkConn) {
	xl := pw.xl
	pw.mu.RLock()
	pxy := pw.pxy
	pw.mu.RUnlock()
	if pxy != nil {
		xl.Debug("start a new work connection, localAddr: %s remoteAddr: %s", workConn.LocalAddr().String(), workConn.RemoteAddr().String())
		go pxy.InWorkConn(workConn, m)
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
