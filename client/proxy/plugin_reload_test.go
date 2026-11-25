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

//go:build !frps

package proxy

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	plugin "github.com/fatedier/frp/pkg/plugin/client"
	"github.com/fatedier/frp/pkg/util/xlog"
)

type reloadError struct {
	err error
}

type testReloadPlugin struct {
	calls  atomic.Int32
	state  atomic.Pointer[reloadError]
	reload func() error
}

func (p *testReloadPlugin) Name() string                                   { return "test" }
func (p *testReloadPlugin) Handle(context.Context, *plugin.ConnectionInfo) {}
func (p *testReloadPlugin) Close() error                                   { return nil }

func (p *testReloadPlugin) Reload() error {
	p.calls.Add(1)
	err := p.reload()
	p.state.Store(&reloadError{err: err})
	return err
}

func (p *testReloadPlugin) ReloadError() error {
	if s := p.state.Load(); s != nil {
		return s.err
	}
	return nil
}

func reloadFixture(p *testReloadPlugin) (*Manager, *Wrapper, *BaseProxy) {
	base := &BaseProxy{proxyPlugin: p}
	pw := &Wrapper{
		WorkingStatus: WorkingStatus{Name: "same", Phase: ProxyPhaseRunning},
		pxy:           &GeneralTCPProxy{BaseProxy: base},
		xl:            xlog.New(),
	}
	return &Manager{proxies: map[string]*Wrapper{"same": pw}}, pw, base
}

func TestPluginSnapshotSkipsReplacedAndStopped(t *testing.T) {
	p := &testReloadPlugin{reload: func() error { return nil }}
	pm, pw, base := reloadFixture(p)
	run := pm.PreparePluginReload()
	q := &testReloadPlugin{reload: func() error { return nil }}
	pw.mu.Lock()
	base.proxyPlugin = q
	pw.mu.Unlock()
	run()
	require.Zero(t, p.calls.Load())
	require.Zero(t, q.calls.Load())
	run = pm.PreparePluginReload()
	pw.mu.Lock()
	pw.Phase = ProxyPhaseClosed
	pw.mu.Unlock()
	run()
	require.Zero(t, q.calls.Load())
}

func TestPluginErrorOwnershipAndLockIsolation(t *testing.T) {
	started, release, done := make(chan struct{}), make(chan struct{}), make(chan struct{})
	p := &testReloadPlugin{reload: func() error {
		close(started)
		<-release
		return errors.New("bad certificate")
	}}
	pm, pw, base := reloadFixture(p)
	other := &testReloadPlugin{reload: func() error { return nil }}
	pm2, _, _ := reloadFixture(other)
	run := pm.PreparePluginReload()
	releaseReload := sync.OnceFunc(func() { close(release) })
	t.Cleanup(func() {
		releaseReload()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Error("reload did not finish")
		}
	})
	go func() {
		run()
		close(done)
	}()
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("not started")
	}
	require.True(t, pm.mu.TryLock())
	pm.mu.Unlock()
	require.True(t, pw.mu.TryLock())
	pw.mu.Unlock()
	require.Empty(t, pw.GetStatus().Err)
	require.Zero(t, other.calls.Load(), "same name in another manager must be isolated")
	pm2.PreparePluginReload()()
	require.EqualValues(t, 1, other.calls.Load())
	q := &testReloadPlugin{reload: func() error { return nil }}
	pw.mu.Lock()
	base.proxyPlugin = q
	pw.mu.Unlock()
	releaseReload()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("not done")
	}
	require.Error(t, p.ReloadError())
	require.Empty(t, pw.GetStatus().Err, "retired plugin errors must not attach to its replacement")
	q.state.Store(&reloadError{err: errors.New("new plugin error")})
	require.Contains(t, pw.GetStatus().Err, "new plugin error")
	require.Empty(t, pw.Err, "plugin state must not overwrite lifecycle errors")
	pw.mu.Lock()
	pw.Err = "lifecycle"
	pw.mu.Unlock()
	require.Contains(t, pw.GetStatus().Err, "lifecycle; reload plugin")
	require.NoError(t, q.Reload())
	require.Equal(t, "lifecycle", pw.GetStatus().Err)
}
