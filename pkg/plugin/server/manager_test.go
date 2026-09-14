// Copyright 2019 fatedier, fatedier@gmail.com
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

package server

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	goliblog "github.com/fatedier/golib/log"
	"github.com/stretchr/testify/require"

	"github.com/fatedier/frp/pkg/msg"
	frplog "github.com/fatedier/frp/pkg/util/log"
)

type testPlugin struct {
	name    string
	ops     map[string]bool
	handler func(context.Context, string, any) (*Response, any, error)
}

// Log-capturing subtests serialize global logger swaps; do not use t.Parallel.
var logCaptureMu sync.Mutex

type logCapture struct {
	bytes.Buffer
	levels []goliblog.Level
}

func (p testPlugin) Name() string {
	return p.name
}

func (p testPlugin) IsSupport(op string) bool {
	return p.ops[op]
}

func (p testPlugin) Handle(ctx context.Context, op string, content any) (*Response, any, error) {
	return p.handler(ctx, op, content)
}

func (w *logCapture) WriteLog(p []byte, level goliblog.Level, _ time.Time) (int, error) {
	w.levels = append(w.levels, level)
	return w.Write(p)
}

func captureLogOutput(t *testing.T) *logCapture {
	t.Helper()

	logCaptureMu.Lock()
	logOutput := &logCapture{}
	oldLogger := frplog.Logger
	frplog.Logger = goliblog.New(
		goliblog.WithOutput(logOutput),
		goliblog.WithLevel(goliblog.TraceLevel),
		goliblog.WithCaller(false),
	)
	t.Cleanup(func() {
		frplog.Logger = oldLogger
		logCaptureMu.Unlock()
	})
	return logOutput
}

var mutablePluginOps = []struct {
	name string
	op   string
}{
	{name: "login", op: OpLogin},
	{name: "new proxy", op: OpNewProxy},
	{name: "ping", op: OpPing},
	{name: "new work conn", op: OpNewWorkConn},
	{name: "new user conn", op: OpNewUserConn},
}

func callMutableWithUser(m *Manager, op string, user string) (string, error) {
	switch op {
	case OpLogin:
		got, err := m.Login(&LoginContent{Login: msg.Login{User: user}})
		if got == nil {
			return "", err
		}
		return got.User, err
	case OpNewProxy:
		got, err := m.NewProxy(&NewProxyContent{User: UserInfo{User: user}})
		if got == nil {
			return "", err
		}
		return got.User.User, err
	case OpPing:
		got, err := m.Ping(&PingContent{User: UserInfo{User: user}})
		if got == nil {
			return "", err
		}
		return got.User.User, err
	case OpNewWorkConn:
		got, err := m.NewWorkConn(&NewWorkConnContent{User: UserInfo{User: user}})
		if got == nil {
			return "", err
		}
		return got.User.User, err
	case OpNewUserConn:
		got, err := m.NewUserConn(&NewUserConnContent{User: UserInfo{User: user}})
		if got == nil {
			return "", err
		}
		return got.User.User, err
	default:
		panic("unsupported mutable op: " + op)
	}
}

func mutableUser(t *testing.T, op string, content any) string {
	t.Helper()

	switch op {
	case OpLogin:
		return content.(LoginContent).User
	case OpNewProxy:
		return content.(NewProxyContent).User.User
	case OpPing:
		return content.(PingContent).User.User
	case OpNewWorkConn:
		return content.(NewWorkConnContent).User.User
	case OpNewUserConn:
		return content.(NewUserConnContent).User.User
	default:
		t.Fatalf("unsupported mutable op: %s", op)
		return ""
	}
}

func mutateMutableContent(t *testing.T, op string, content any, user string) any {
	t.Helper()

	switch op {
	case OpLogin:
		got := content.(LoginContent)
		got.User = user
		return &got
	case OpNewProxy:
		got := content.(NewProxyContent)
		got.User.User = user
		return &got
	case OpPing:
		got := content.(PingContent)
		got.User.User = user
		return &got
	case OpNewWorkConn:
		got := content.(NewWorkConnContent)
		got.User.User = user
		return &got
	case OpNewUserConn:
		got := content.(NewUserConnContent)
		got.User.User = user
		return &got
	default:
		t.Fatalf("unsupported mutable op: %s", op)
		return nil
	}
}

func TestManagerMutableContentAcrossOps(t *testing.T) {
	for _, tt := range mutablePluginOps {
		t.Run(tt.name, func(t *testing.T) {
			m := NewManager()
			m.Register(testPlugin{
				name: "mutate",
				ops:  map[string]bool{tt.op: true},
				handler: func(ctx context.Context, op string, content any) (*Response, any, error) {
					if op != tt.op {
						t.Fatalf("unexpected op: %s", op)
					}
					if GetReqidFromContext(ctx) == "" {
						t.Fatal("expected request id in context")
					}
					if got := mutableUser(t, tt.op, content); got != "initial" {
						t.Fatalf("expected initial user, got %q", got)
					}
					return &Response{Unchange: false}, mutateMutableContent(t, tt.op, content, "mutated"), nil
				},
			})
			m.Register(testPlugin{
				name: "observe",
				ops:  map[string]bool{tt.op: true},
				handler: func(ctx context.Context, op string, content any) (*Response, any, error) {
					if op != tt.op {
						t.Fatalf("unexpected op: %s", op)
					}
					if GetReqidFromContext(ctx) == "" {
						t.Fatal("expected request id in context")
					}
					if got := mutableUser(t, tt.op, content); got != "mutated" {
						t.Fatalf("expected mutated user, got %q", got)
					}
					return &Response{Unchange: true}, mutateMutableContent(t, tt.op, content, "ignored"), nil
				},
			})

			got, err := callMutableWithUser(m, tt.op, "initial")
			if err != nil {
				t.Fatalf("mutable op failed: %v", err)
			}
			if got != "mutated" {
				t.Fatalf("expected mutated user, got %q", got)
			}
		})
	}
}

func TestManagerMutableContentRejectStopsChain(t *testing.T) {
	m := NewManager()

	var called bool
	m.Register(testPlugin{
		name: "reject",
		ops:  map[string]bool{OpPing: true},
		handler: func(context.Context, string, any) (*Response, any, error) {
			return &Response{Reject: true, RejectReason: "blocked"}, nil, nil
		},
	})
	m.Register(testPlugin{
		name: "unused",
		ops:  map[string]bool{OpPing: true},
		handler: func(context.Context, string, any) (*Response, any, error) {
			called = true
			return &Response{Unchange: true}, nil, nil
		},
	})

	got, err := m.Ping(&PingContent{})
	if err == nil {
		t.Fatal("expected reject error")
	}
	if got != nil {
		t.Fatalf("expected no returned content, got %#v", got)
	}
	if err.Error() != "blocked" {
		t.Fatalf("unexpected error: %v", err)
	}
	if called {
		t.Fatal("expected plugin chain to stop after reject")
	}
}

func TestManagerNewWorkConnPreservesMetadataBetweenPlugins(t *testing.T) {
	for _, tc := range []struct {
		name string
		id   uint64
		kind string
	}{
		{name: "typed", id: 7, kind: msg.WorkConnTypeReserve},
		{name: "legacy"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := NewManager()
			observations := 0
			for range 2 {
				// Model an old plugin replacing the entire content, including
				// intentionally omitted ordinary fields. Only metadata is restored.
				partial := &NewWorkConnContent{NewWorkConn: msg.NewWorkConn{RunID: "mutated", Timestamp: 42}}
				m.Register(testPlugin{
					name: "partial", ops: map[string]bool{OpNewWorkConn: true},
					handler: func(context.Context, string, any) (*Response, any, error) {
						return &Response{Unchange: false}, partial, nil
					},
				})
				m.Register(testPlugin{
					name: "policy", ops: map[string]bool{OpNewWorkConn: true},
					handler: func(_ context.Context, _ string, raw any) (*Response, any, error) {
						content := raw.(NewWorkConnContent)
						observations++
						if content.ControlID != tc.id || content.WorkConnType != tc.kind {
							return &Response{Reject: true, RejectReason: "lost scheduling metadata"}, nil, nil
						}
						require.Equal(t, "mutated", content.RunID)
						require.Equal(t, int64(42), content.Timestamp)
						require.Empty(t, content.User.User, "ordinary omitted fields must not be merged back")
						require.Zero(t, partial.ControlID, "do not mutate the plugin's response")
						// Unchange must ignore even an invalid returned generation.
						content.ControlID = tc.id + 1
						content.WorkConnType = "ignored"
						return &Response{Unchange: true}, &content, nil
					},
				})
			}
			got, err := m.NewWorkConn(&NewWorkConnContent{
				User:        UserInfo{User: "original"},
				NewWorkConn: msg.NewWorkConn{RunID: "original", ControlID: tc.id, WorkConnType: tc.kind},
			})
			require.NoError(t, err)
			require.Equal(t, 2, observations)
			require.Equal(t, tc.id, got.ControlID)
			require.Equal(t, tc.kind, got.WorkConnType)
			require.Equal(t, "mutated", got.RunID)
		})
	}
}

func TestManagerNewWorkConnMetadataValidationPreservesPluginContract(t *testing.T) {
	for _, tc := range []struct {
		name       string
		initialID  uint64
		returnedID uint64
		response   Response
		pluginErr  error
		wantErr    string
		wantIDErr  bool
	}{
		{name: "changed generation", initialID: 7, returnedID: 8, wantErr: "from 7 to 8", wantIDErr: true},
		{name: "legacy cannot inject generation", returnedID: 8, wantErr: "from 0 to 8", wantIDErr: true},
		{name: "same generation", initialID: 7, returnedID: 7},
		{name: "omitted generation", initialID: 7},
		{name: "unchange ignores generation rewrite", initialID: 7, returnedID: 8, response: Response{Unchange: true}},
		{
			name: "reject stops before validation", initialID: 7, returnedID: 8,
			response: Response{Reject: true, RejectReason: "policy rejection"}, wantErr: "policy rejection",
		},
		{
			name: "error stops before validation", initialID: 7, returnedID: 8,
			pluginErr: errors.New("unavailable"), wantErr: "send NewWorkConn request to plugin error",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := NewManager()
			m.Register(testPlugin{
				name: "mutate", ops: map[string]bool{OpNewWorkConn: true},
				handler: func(context.Context, string, any) (*Response, any, error) {
					return &tc.response, &NewWorkConnContent{NewWorkConn: msg.NewWorkConn{
						ControlID: tc.returnedID, WorkConnType: msg.WorkConnTypeDemand,
					}}, tc.pluginErr
				},
			})
			called := false
			m.Register(testPlugin{
				name: "observe", ops: map[string]bool{OpNewWorkConn: true},
				handler: func(_ context.Context, _ string, raw any) (*Response, any, error) {
					called = true
					content := raw.(NewWorkConnContent)
					require.Equal(t, tc.initialID, content.ControlID)
					require.Equal(t, msg.WorkConnTypeReserve, content.WorkConnType)
					return &Response{Unchange: true}, nil, nil
				},
			})
			got, err := m.NewWorkConn(&NewWorkConnContent{NewWorkConn: msg.NewWorkConn{
				ControlID: tc.initialID, WorkConnType: msg.WorkConnTypeReserve,
			}})
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
				require.Nil(t, got)
				require.False(t, called)
			} else {
				require.NoError(t, err)
				require.NotNil(t, got)
				require.True(t, called)
			}
			require.Equal(t, tc.wantIDErr, errors.Is(err, ErrNewWorkConnControlIDChanged))
		})
	}
}

func TestManagerNewWorkConnChangedContentRequiresPointer(t *testing.T) {
	m := NewManager()
	m.Register(testPlugin{
		name: "invalid", ops: map[string]bool{OpNewWorkConn: true},
		handler: func(context.Context, string, any) (*Response, any, error) {
			return &Response{Unchange: false}, NewWorkConnContent{}, nil
		},
	})
	require.Panics(t, func() { _, _ = m.NewWorkConn(&NewWorkConnContent{}) })
}

func TestManagerMutableContentPluginErrorLogLevel(t *testing.T) {
	tests := []struct {
		name  string
		op    string
		level goliblog.Level
	}{
		{name: "default warning", op: OpLogin, level: goliblog.WarnLevel},
		{name: "new user conn info", op: OpNewUserConn, level: goliblog.InfoLevel},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logOutput := captureLogOutput(t)
			m := NewManager()
			m.Register(testPlugin{
				name: "error",
				ops:  map[string]bool{tt.op: true},
				handler: func(context.Context, string, any) (*Response, any, error) {
					return nil, nil, errors.New("boom")
				},
			})

			_, err := callMutableWithUser(m, tt.op, "initial")
			if err == nil {
				t.Fatal("expected plugin error")
			}
			if want := "send " + tt.op + " request to plugin error"; err.Error() != want {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(logOutput.levels) != 1 || logOutput.levels[0] != tt.level {
				t.Fatalf("expected log level %v, got %v in %q", tt.level, logOutput.levels, logOutput.String())
			}
		})
	}
}

func TestManagerCloseProxyAggregatesErrors(t *testing.T) {
	logOutput := captureLogOutput(t)
	m := NewManager()

	for _, name := range []string{"first", "second"} {
		m.Register(testPlugin{
			name: name,
			ops:  map[string]bool{OpCloseProxy: true},
			handler: func(ctx context.Context, op string, content any) (*Response, any, error) {
				if GetReqidFromContext(ctx) == "" {
					t.Fatal("expected request id in context")
				}
				if op != OpCloseProxy {
					t.Fatalf("unexpected op: %s", op)
				}
				return nil, nil, errors.New(name + " error")
			},
		})
	}

	err := m.CloseProxy(&CloseProxyContent{})
	if err == nil {
		t.Fatal("expected close proxy error")
	}
	if !strings.HasPrefix(err.Error(), "send CloseProxy request to plugin errors: ") {
		t.Fatalf("unexpected close proxy error prefix: %v", err)
	}
	if !strings.Contains(err.Error(), "[first]: first error") || !strings.Contains(err.Error(), "[second]: second error") {
		t.Fatalf("missing aggregated errors: %v", err)
	}
	if len(logOutput.levels) != 2 {
		t.Fatalf("expected two warning logs, got %v", logOutput.levels)
	}
	for _, level := range logOutput.levels {
		if level != goliblog.WarnLevel {
			t.Fatalf("expected warning log level, got %v", logOutput.levels)
		}
	}
}
