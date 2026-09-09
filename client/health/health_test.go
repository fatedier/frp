// Copyright 2026 The frp Authors
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
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	v1 "github.com/fatedier/frp/pkg/config/v1"
)

type tcpHealthBackend struct {
	listener net.Listener
	accepted chan struct{}
	done     chan struct{}
}

func newTCPHealthBackend(t *testing.T, addr string, accepted chan struct{}) *tcpHealthBackend {
	listener, err := net.Listen("tcp", addr)
	require.NoError(t, err)

	backend := &tcpHealthBackend{
		listener: listener,
		accepted: accepted,
		done:     make(chan struct{}),
	}
	go func() {
		defer close(backend.done)
		for {
			conn, err := backend.listener.Accept()
			if err != nil {
				return
			}
			_ = conn.Close()
			select {
			case backend.accepted <- struct{}{}:
			default:
			}
		}
	}()
	return backend
}

func (backend *tcpHealthBackend) Close() {
	_ = backend.listener.Close()
	<-backend.done
}

func TestMonitorConsecutiveFailureWindows(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*testing.T, func(), func()) (*Monitor, func(bool))
	}{
		{
			name: "HTTP",
			setup: func(t *testing.T, normalFn, failedFn func()) (*Monitor, func(bool)) {
				var healthy atomic.Bool
				healthy.Store(true)
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					if !healthy.Load() {
						w.WriteHeader(http.StatusServiceUnavailable)
					}
				}))
				t.Cleanup(server.Close)

				monitor := NewMonitor(
					context.Background(),
					v1.HealthCheckConfig{
						Type:            "http",
						Path:            "/health",
						TimeoutSeconds:  1,
						IntervalSeconds: 1,
						MaxFailed:       3,
					},
					strings.TrimPrefix(server.URL, "http://"),
					normalFn,
					failedFn,
				)
				return monitor, healthy.Store
			},
		},
		{
			name: "TCP",
			setup: func(t *testing.T, normalFn, failedFn func()) (*Monitor, func(bool)) {
				listener, err := net.Listen("tcp", "127.0.0.1:0")
				require.NoError(t, err)
				t.Cleanup(func() { _ = listener.Close() })
				go func() {
					for {
						conn, err := listener.Accept()
						if err != nil {
							return
						}
						_ = conn.Close()
					}
				}()

				monitor := NewMonitor(
					context.Background(),
					v1.HealthCheckConfig{
						Type:            "tcp",
						TimeoutSeconds:  1,
						IntervalSeconds: 1,
						MaxFailed:       3,
					},
					listener.Addr().String(),
					normalFn,
					failedFn,
				)
				healthyAddr := monitor.addr
				return monitor, func(healthy bool) {
					if healthy {
						monitor.addr = healthyAddr
					} else {
						monitor.addr = "127.0.0.1:0"
					}
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var events []string
			monitor, setHealthy := test.setup(
				t,
				func() { events = append(events, "normal") },
				func() { events = append(events, "failed") },
			)
			t.Cleanup(monitor.Stop)

			runCheck := func(healthy bool) {
				t.Helper()
				setHealthy(healthy)
				ctx, cancel := context.WithTimeout(monitor.ctx, time.Second)
				err := monitor.doCheck(ctx)
				cancel()
				if healthy {
					require.NoError(t, err)
				} else {
					require.Error(t, err)
				}
				monitor.handleCheckResult(err)
			}

			runCheck(true)
			require.True(t, monitor.statusOK)
			require.Zero(t, monitor.failedTimes)
			require.Equal(t, []string{"normal"}, events)

			runCheck(false)
			runCheck(false)
			require.True(t, monitor.statusOK)
			require.Equal(t, uint64(2), monitor.failedTimes)
			require.Equal(t, []string{"normal"}, events)

			runCheck(true)
			require.True(t, monitor.statusOK)
			require.Zero(t, monitor.failedTimes)
			require.Equal(t, []string{"normal"}, events)

			runCheck(false)
			require.True(t, monitor.statusOK)
			require.Equal(t, uint64(1), monitor.failedTimes)
			require.Equal(t, []string{"normal"}, events)

			runCheck(false)
			require.True(t, monitor.statusOK)
			require.Equal(t, uint64(2), monitor.failedTimes)

			runCheck(false)
			require.False(t, monitor.statusOK)
			require.Equal(t, uint64(3), monitor.failedTimes)
			require.Equal(t, []string{"normal", "failed"}, events)

			runCheck(true)
			require.True(t, monitor.statusOK)
			require.Zero(t, monitor.failedTimes)
			require.Equal(t, []string{"normal", "failed", "normal"}, events)
		})
	}
}

func TestMonitorWorkerProcessesResults(t *testing.T) {
	requestReady := make(chan struct{}, 1)
	responses := make(chan int)
	requestCanceled := make(chan struct{})
	var cancelOnce sync.Once
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestReady <- struct{}{}
		select {
		case code := <-responses:
			w.WriteHeader(code)
		case <-r.Context().Done():
			cancelOnce.Do(func() { close(requestCanceled) })
		}
	}))
	t.Cleanup(server.Close)

	events := make(chan string, 3)
	monitor := NewMonitor(
		context.Background(),
		v1.HealthCheckConfig{
			Type:           "http",
			Path:           "/health",
			TimeoutSeconds: 1,
			MaxFailed:      3,
		},
		strings.TrimPrefix(server.URL, "http://"),
		func() { events <- "normal" },
		func() { events <- "failed" },
	)
	monitor.interval = 0
	t.Cleanup(monitor.Stop)

	awaitRequest := func() {
		t.Helper()
		select {
		case <-requestReady:
		case <-time.After(time.Second):
			t.Fatal("health check request did not start")
		}
	}
	awaitEvent := func(want string) {
		t.Helper()
		select {
		case got := <-events:
			require.Equal(t, want, got)
		case <-time.After(time.Second):
			t.Fatalf("health check callback %q was not called", want)
		}
	}
	respond := func(code int) {
		t.Helper()
		responses <- code
	}

	monitor.Start()

	awaitRequest()
	respond(http.StatusOK)
	awaitEvent("normal")

	awaitRequest()
	require.True(t, monitor.statusOK)
	require.Zero(t, monitor.failedTimes)
	respond(http.StatusServiceUnavailable)

	awaitRequest()
	require.True(t, monitor.statusOK)
	require.Equal(t, uint64(1), monitor.failedTimes)
	respond(http.StatusServiceUnavailable)

	awaitRequest()
	require.True(t, monitor.statusOK)
	require.Equal(t, uint64(2), monitor.failedTimes)
	respond(http.StatusOK)

	awaitRequest()
	require.True(t, monitor.statusOK)
	require.Zero(t, monitor.failedTimes)
	respond(http.StatusServiceUnavailable)

	awaitRequest()
	require.True(t, monitor.statusOK)
	require.Equal(t, uint64(1), monitor.failedTimes)
	respond(http.StatusServiceUnavailable)

	awaitRequest()
	require.True(t, monitor.statusOK)
	require.Equal(t, uint64(2), monitor.failedTimes)
	respond(http.StatusServiceUnavailable)
	awaitEvent("failed")

	awaitRequest()
	require.False(t, monitor.statusOK)
	require.Equal(t, uint64(3), monitor.failedTimes)
	respond(http.StatusOK)
	awaitEvent("normal")

	awaitRequest()
	require.True(t, monitor.statusOK)
	require.Zero(t, monitor.failedTimes)
	monitor.Stop()
	select {
	case <-requestCanceled:
	case <-time.After(time.Second):
		t.Fatal("health check request context was not canceled")
	}
}

func TestMonitorTCPWorkerProcessesResults(t *testing.T) {
	initialAccepted := make(chan struct{}, 1)
	initialBackend := newTCPHealthBackend(t, "127.0.0.1:0", initialAccepted)
	addr := initialBackend.listener.Addr().String()
	recoveryAccepted := make(chan struct{}, 1)

	type workerStatus struct {
		failedTimes uint64
		statusOK    bool
	}
	normalCallbacks := make(chan workerStatus, 2)
	failedCallbacks := make(chan workerStatus, 2)
	timerReady := make(chan chan time.Time, 16)
	var monitor *Monitor
	monitor = NewMonitor(
		context.Background(),
		v1.HealthCheckConfig{
			Type:           "tcp",
			TimeoutSeconds: 1,
			MaxFailed:      3,
		},
		addr,
		func() {
			normalCallbacks <- workerStatus{failedTimes: monitor.failedTimes, statusOK: monitor.statusOK}
		},
		func() {
			failedCallbacks <- workerStatus{failedTimes: monitor.failedTimes, statusOK: monitor.statusOK}
		},
	)
	monitor.interval = 0
	monitor.timerFactory = func(time.Duration) (<-chan time.Time, func()) {
		timer := make(chan time.Time, 1)
		timerReady <- timer
		return timer, func() {}
	}
	recoveryBackend := (*tcpHealthBackend)(nil)
	t.Cleanup(func() {
		monitor.Stop()
		initialBackend.Close()
		if recoveryBackend != nil {
			recoveryBackend.Close()
		}
	})

	awaitTimer := func() chan time.Time {
		t.Helper()
		select {
		case timer := <-timerReady:
			return timer
		case <-time.After(time.Second):
			t.Fatal("TCP worker did not reach the interval barrier")
			return nil
		}
	}
	awaitStatus := func(ch <-chan workerStatus, want workerStatus, message string) {
		t.Helper()
		select {
		case got := <-ch:
			require.Equal(t, want, got)
		case <-time.After(time.Second):
			t.Fatal(message)
		}
	}

	monitor.Start()
	select {
	case <-initialAccepted:
	case <-time.After(time.Second):
		t.Fatal("TCP health check did not reach the initial backend")
	}
	awaitStatus(normalCallbacks, workerStatus{failedTimes: 0, statusOK: true}, "TCP worker did not report the initial success")

	initialTimer := awaitTimer()
	initialBackend.Close()
	initialTimer <- time.Now()

	firstFailureTimer := awaitTimer()
	require.Equal(t, uint64(1), monitor.failedTimes)
	require.True(t, monitor.statusOK)
	firstFailureTimer <- time.Now()

	secondFailureTimer := awaitTimer()
	require.Equal(t, uint64(2), monitor.failedTimes)
	require.True(t, monitor.statusOK)
	secondFailureTimer <- time.Now()

	awaitStatus(failedCallbacks, workerStatus{failedTimes: 3, statusOK: false}, "TCP worker did not report the third failed health check")
	thirdFailureTimer := awaitTimer()

	recoveryBackend = newTCPHealthBackend(t, addr, recoveryAccepted)
	thirdFailureTimer <- time.Now()
	select {
	case <-recoveryAccepted:
	case <-time.After(time.Second):
		t.Fatal("TCP health check did not reach the recovery backend")
	}
	awaitStatus(normalCallbacks, workerStatus{failedTimes: 0, statusOK: true}, "TCP worker did not report recovery")

	recoveryTimer := awaitTimer()
	recoveryBackend.Close()
	recoveryTimer <- time.Now()

	firstRecoveryFailureTimer := awaitTimer()
	require.Equal(t, uint64(1), monitor.failedTimes)
	require.True(t, monitor.statusOK)
	firstRecoveryFailureTimer <- time.Now()

	secondRecoveryFailureTimer := awaitTimer()
	require.Equal(t, uint64(2), monitor.failedTimes)
	require.True(t, monitor.statusOK)
	secondRecoveryFailureTimer <- time.Now()

	awaitStatus(failedCallbacks, workerStatus{failedTimes: 3, statusOK: false}, "TCP worker did not report the third failed health check after recovery")
	monitor.Stop()
	select {
	case <-monitor.ctx.Done():
	case <-time.After(time.Second):
		t.Fatal("TCP worker did not stop after cancellation")
	}
}

func TestMonitorMaxFailedOne(t *testing.T) {
	var events []string
	monitor := NewMonitor(
		context.Background(),
		v1.HealthCheckConfig{Type: "tcp", MaxFailed: 1},
		"",
		func() { events = append(events, "normal") },
		func() { events = append(events, "failed") },
	)
	t.Cleanup(monitor.Stop)

	checkErr := errors.New("health check failed")
	monitor.handleCheckResult(nil)
	monitor.handleCheckResult(checkErr)
	require.Equal(t, []string{"normal", "failed"}, events)
	require.False(t, monitor.statusOK)
	require.Equal(t, uint64(1), monitor.failedTimes)

	monitor.handleCheckResult(checkErr)
	require.Equal(t, []string{"normal", "failed"}, events)

	monitor.handleCheckResult(nil)
	monitor.handleCheckResult(checkErr)
	require.Equal(t, []string{"normal", "failed", "normal", "failed"}, events)
}

func TestMonitorStopCancelsWork(t *testing.T) {
	t.Run("in-flight check", func(t *testing.T) {
		requestStarted := make(chan struct{})
		requestCanceled := make(chan struct{})
		server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			close(requestStarted)
			<-r.Context().Done()
			close(requestCanceled)
		}))
		t.Cleanup(server.Close)

		monitor := NewMonitor(
			context.Background(),
			v1.HealthCheckConfig{Type: "http", Path: "/health"},
			strings.TrimPrefix(server.URL, "http://"),
			func() {},
			func() {},
		)
		t.Cleanup(monitor.Stop)
		monitor.Start()

		select {
		case <-requestStarted:
		case <-time.After(time.Second):
			t.Fatal("health check request did not start")
		}
		monitor.Stop()
		select {
		case <-requestCanceled:
		case <-time.After(time.Second):
			t.Fatal("health check request context was not canceled")
		}
	})

	t.Run("interval wait", func(t *testing.T) {
		monitor := NewMonitor(context.Background(), v1.HealthCheckConfig{Type: "tcp"}, "", nil, nil)
		monitor.interval = time.Hour
		waitResult := make(chan bool, 1)
		go func() {
			waitResult <- monitor.waitForNextCheck()
		}()

		monitor.Stop()
		select {
		case shouldContinue := <-waitResult:
			require.False(t, shouldContinue)
		case <-time.After(time.Second):
			t.Fatal("interval wait did not stop after cancellation")
		}
	})
}

func TestMonitorTimerCancellationWinsWhenBothReady(t *testing.T) {
	monitor := NewMonitor(
		context.Background(),
		v1.HealthCheckConfig{Type: "tcp"},
		"",
		nil,
		nil,
	)

	timerReady := make(chan time.Time, 1)
	timerReady <- time.Now()
	var timerStopped atomic.Bool
	monitor.timerFactory = func(time.Duration) (<-chan time.Time, func()) {
		return timerReady, func() { timerStopped.Store(true) }
	}
	monitor.Stop()

	// Both channels are ready before waitForNextCheck starts. Whichever select
	// branch is chosen must return false; the timer branch must re-check ctx.
	require.False(t, monitor.waitForNextCheck())
	require.True(t, timerStopped.Load())
}
