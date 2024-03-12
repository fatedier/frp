// Copyright 2018 fatedier, fatedier@gmail.com
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package health

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/util/xlog"
)

var ErrHealthCheckType = errors.New("error health check type")

type Monitor struct {
	checkType      string
	interval       time.Duration
	timeout        time.Duration
	maxFailedTimes int

	// For tcp
	addr string

	// For http
	url string

	failedTimes    uint64
	statusOK       bool
	statusNormalFn func()
	statusFailedFn func()

	ctx    context.Context
	cancel context.CancelFunc
}

func NewMonitor(ctx context.Context, cfg v1.HealthCheckConfig, addr string,
	statusNormalFn func(), statusFailedFn func(),
) *Monitor {
	if cfg.IntervalSeconds <= 0 {
		cfg.IntervalSeconds = 10
	}
	if cfg.TimeoutSeconds <= 0 {
		cfg.TimeoutSeconds = 3
	}
	if cfg.MaxFailed <= 0 {
		cfg.MaxFailed = 1
	}
	newctx, cancel := context.WithCancel(ctx)

	var url string
	if cfg.Type == "http" && cfg.Path != "" {
		s := "http://" + addr
		if !strings.HasPrefix(cfg.Path, "/") {
			s += "/"
		}
		url = s + cfg.Path
	}
	return &Monitor{
		checkType:      cfg.Type,
		interval:       time.Duration(cfg.IntervalSeconds) * time.Second,
		timeout:        time.Duration(cfg.TimeoutSeconds) * time.Second,
		maxFailedTimes: cfg.MaxFailed,
		addr:           addr,
		url:            url,
		statusOK:       false,
		statusNormalFn: statusNormalFn,
		statusFailedFn: statusFailedFn,
		ctx:            newctx,
		cancel:         cancel,
	}
}

func (monitor *Monitor) Start() {
	go monitor.checkWorker()
}

func (monitor *Monitor) Stop() {
	monitor.cancel()
}

func (monitor *Monitor) checkWorker() {
	xl := xlog.FromContextSafe(monitor.ctx)
	for {
		doCtx, cancel := context.WithDeadline(monitor.ctx, time.Now().Add(monitor.timeout))
		err := monitor.doCheck(doCtx)

		// check if this monitor has been closed
		select {
		case <-monitor.ctx.Done():
			cancel()
			return
		default:
			cancel()
		}

		if err == nil {
			xl.Tracef("do one health check success")
			if !monitor.statusOK && monitor.statusNormalFn != nil {
				xl.Infof("health check status change to success")
				monitor.statusOK = true
				monitor.statusNormalFn()
			}
		} else {
			xl.Warnf("do one health check failed: %v", err)
			monitor.failedTimes++
			if monitor.statusOK && int(monitor.failedTimes) >= monitor.maxFailedTimes && monitor.statusFailedFn != nil {
				xl.Warnf("health check status change to failed")
				monitor.statusOK = false
				monitor.statusFailedFn()
			}
		}

		time.Sleep(monitor.interval)
	}
}

func (monitor *Monitor) doCheck(ctx context.Context) error {
	switch monitor.checkType {
	case "tcp":
		return monitor.doTCPCheck(ctx)
	case "http":
		return monitor.doHTTPCheck(ctx)
	default:
		return ErrHealthCheckType
	}
}

func (monitor *Monitor) doTCPCheck(ctx context.Context) error {
	// if tcp address is not specified, always return nil
	if monitor.addr == "" {
		return nil
	}

	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", monitor.addr)
	if err != nil {
		return err
	}
	conn.Close()
	return nil
}

func (monitor *Monitor) doHTTPCheck(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", monitor.url, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("do http health check, StatusCode is [%d] not 2xx", resp.StatusCode)
	}
	return nil
}
