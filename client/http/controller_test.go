package http

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"

	"github.com/fatedier/frp/client/configmgmt"
	"github.com/fatedier/frp/client/http/model"
	"github.com/fatedier/frp/client/proxy"
	v1 "github.com/fatedier/frp/pkg/config/v1"
	httppkg "github.com/fatedier/frp/pkg/util/http"
)

type fakeConfigManager struct {
	reloadFromFileFn      func(strict bool) error
	readConfigFileFn      func() (string, error)
	writeConfigFileFn     func(content []byte) error
	getProxyStatusFn      func() []*proxy.WorkingStatus
	isStoreProxyEnabledFn func(name string) bool
	storeEnabledFn        func() bool
	getProxyConfigFn      func(name string) (v1.ProxyConfigurer, bool)
	getVisitorConfigFn    func(name string) (v1.VisitorConfigurer, bool)

	listStoreProxiesFn  func() ([]v1.ProxyConfigurer, error)
	getStoreProxyFn     func(name string) (v1.ProxyConfigurer, error)
	createStoreProxyFn  func(cfg v1.ProxyConfigurer) (v1.ProxyConfigurer, error)
	updateStoreProxyFn  func(name string, cfg v1.ProxyConfigurer) (v1.ProxyConfigurer, error)
	deleteStoreProxyFn  func(name string) error
	listStoreVisitorsFn func() ([]v1.VisitorConfigurer, error)
	getStoreVisitorFn   func(name string) (v1.VisitorConfigurer, error)
	createStoreVisitFn  func(cfg v1.VisitorConfigurer) (v1.VisitorConfigurer, error)
	updateStoreVisitFn  func(name string, cfg v1.VisitorConfigurer) (v1.VisitorConfigurer, error)
	deleteStoreVisitFn  func(name string) error
	gracefulCloseFn     func(d time.Duration)
}

func (m *fakeConfigManager) ReloadFromFile(strict bool) error {
	if m.reloadFromFileFn != nil {
		return m.reloadFromFileFn(strict)
	}
	return nil
}

func (m *fakeConfigManager) ReadConfigFile() (string, error) {
	if m.readConfigFileFn != nil {
		return m.readConfigFileFn()
	}
	return "", nil
}

func (m *fakeConfigManager) WriteConfigFile(content []byte) error {
	if m.writeConfigFileFn != nil {
		return m.writeConfigFileFn(content)
	}
	return nil
}

func (m *fakeConfigManager) GetProxyStatus() []*proxy.WorkingStatus {
	if m.getProxyStatusFn != nil {
		return m.getProxyStatusFn()
	}
	return nil
}

func (m *fakeConfigManager) IsStoreProxyEnabled(name string) bool {
	if m.isStoreProxyEnabledFn != nil {
		return m.isStoreProxyEnabledFn(name)
	}
	return false
}

func (m *fakeConfigManager) StoreEnabled() bool {
	if m.storeEnabledFn != nil {
		return m.storeEnabledFn()
	}
	return false
}

func (m *fakeConfigManager) GetProxyConfig(name string) (v1.ProxyConfigurer, bool) {
	if m.getProxyConfigFn != nil {
		return m.getProxyConfigFn(name)
	}
	return nil, false
}

func (m *fakeConfigManager) GetVisitorConfig(name string) (v1.VisitorConfigurer, bool) {
	if m.getVisitorConfigFn != nil {
		return m.getVisitorConfigFn(name)
	}
	return nil, false
}

func (m *fakeConfigManager) ListStoreProxies() ([]v1.ProxyConfigurer, error) {
	if m.listStoreProxiesFn != nil {
		return m.listStoreProxiesFn()
	}
	return nil, nil
}

func (m *fakeConfigManager) GetStoreProxy(name string) (v1.ProxyConfigurer, error) {
	if m.getStoreProxyFn != nil {
		return m.getStoreProxyFn(name)
	}
	return nil, nil
}

func (m *fakeConfigManager) CreateStoreProxy(cfg v1.ProxyConfigurer) (v1.ProxyConfigurer, error) {
	if m.createStoreProxyFn != nil {
		return m.createStoreProxyFn(cfg)
	}
	return cfg, nil
}

func (m *fakeConfigManager) UpdateStoreProxy(name string, cfg v1.ProxyConfigurer) (v1.ProxyConfigurer, error) {
	if m.updateStoreProxyFn != nil {
		return m.updateStoreProxyFn(name, cfg)
	}
	return cfg, nil
}

func (m *fakeConfigManager) DeleteStoreProxy(name string) error {
	if m.deleteStoreProxyFn != nil {
		return m.deleteStoreProxyFn(name)
	}
	return nil
}

func (m *fakeConfigManager) ListStoreVisitors() ([]v1.VisitorConfigurer, error) {
	if m.listStoreVisitorsFn != nil {
		return m.listStoreVisitorsFn()
	}
	return nil, nil
}

func (m *fakeConfigManager) GetStoreVisitor(name string) (v1.VisitorConfigurer, error) {
	if m.getStoreVisitorFn != nil {
		return m.getStoreVisitorFn(name)
	}
	return nil, nil
}

func (m *fakeConfigManager) CreateStoreVisitor(cfg v1.VisitorConfigurer) (v1.VisitorConfigurer, error) {
	if m.createStoreVisitFn != nil {
		return m.createStoreVisitFn(cfg)
	}
	return cfg, nil
}

func (m *fakeConfigManager) UpdateStoreVisitor(name string, cfg v1.VisitorConfigurer) (v1.VisitorConfigurer, error) {
	if m.updateStoreVisitFn != nil {
		return m.updateStoreVisitFn(name, cfg)
	}
	return cfg, nil
}

func (m *fakeConfigManager) DeleteStoreVisitor(name string) error {
	if m.deleteStoreVisitFn != nil {
		return m.deleteStoreVisitFn(name)
	}
	return nil
}

func (m *fakeConfigManager) GracefulClose(d time.Duration) {
	if m.gracefulCloseFn != nil {
		m.gracefulCloseFn(d)
	}
}

func newRawTCPProxyConfig(name string) *v1.TCPProxyConfig {
	return &v1.TCPProxyConfig{
		ProxyBaseConfig: v1.ProxyBaseConfig{
			Name: name,
			Type: "tcp",
			ProxyBackend: v1.ProxyBackend{
				LocalPort: 10080,
			},
		},
	}
}

func TestBuildProxyStatusRespStoreSourceEnabled(t *testing.T) {
	status := &proxy.WorkingStatus{
		Name:       "shared-proxy",
		Type:       "tcp",
		Phase:      proxy.ProxyPhaseRunning,
		RemoteAddr: ":8080",
		Cfg:        newRawTCPProxyConfig("shared-proxy"),
	}

	controller := &Controller{
		serverAddr: "127.0.0.1",
		manager: &fakeConfigManager{
			isStoreProxyEnabledFn: func(name string) bool {
				return name == "shared-proxy"
			},
		},
	}

	resp := controller.buildProxyStatusResp(status)
	if resp.Source != "store" {
		t.Fatalf("unexpected source: %q", resp.Source)
	}
	if resp.RemoteAddr != "127.0.0.1:8080" {
		t.Fatalf("unexpected remote addr: %q", resp.RemoteAddr)
	}
}

func TestReloadErrorMapping(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		expectedCode int
	}{
		{name: "invalid arg", err: fmtError(configmgmt.ErrInvalidArgument, "bad cfg"), expectedCode: http.StatusBadRequest},
		{name: "apply fail", err: fmtError(configmgmt.ErrApplyConfig, "reload failed"), expectedCode: http.StatusInternalServerError},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			controller := &Controller{
				manager: &fakeConfigManager{reloadFromFileFn: func(bool) error { return tc.err }},
			}
			ctx := httppkg.NewContext(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/reload", nil))
			_, err := controller.Reload(ctx)
			if err == nil {
				t.Fatal("expected error")
			}
			assertHTTPCode(t, err, tc.expectedCode)
		})
	}
}

func TestStoreProxyErrorMapping(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		expectedCode int
	}{
		{name: "not found", err: fmtError(configmgmt.ErrNotFound, "not found"), expectedCode: http.StatusNotFound},
		{name: "conflict", err: fmtError(configmgmt.ErrConflict, "exists"), expectedCode: http.StatusConflict},
		{name: "internal", err: errors.New("persist failed"), expectedCode: http.StatusInternalServerError},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			body := []byte(`{"name":"shared-proxy","type":"tcp","tcp":{"localPort":10080}}`)
			req := httptest.NewRequest(http.MethodPut, "/api/store/proxies/shared-proxy", bytes.NewReader(body))
			req = mux.SetURLVars(req, map[string]string{"name": "shared-proxy"})
			ctx := httppkg.NewContext(httptest.NewRecorder(), req)

			controller := &Controller{
				manager: &fakeConfigManager{
					updateStoreProxyFn: func(_ string, _ v1.ProxyConfigurer) (v1.ProxyConfigurer, error) {
						return nil, tc.err
					},
				},
			}

			_, err := controller.UpdateStoreProxy(ctx)
			if err == nil {
				t.Fatal("expected error")
			}
			assertHTTPCode(t, err, tc.expectedCode)
		})
	}
}

func TestStoreVisitorErrorMapping(t *testing.T) {
	body := []byte(`{"name":"shared-visitor","type":"xtcp","xtcp":{"serverName":"server","bindPort":10081,"secretKey":"secret"}}`)
	req := httptest.NewRequest(http.MethodDelete, "/api/store/visitors/shared-visitor", bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"name": "shared-visitor"})
	ctx := httppkg.NewContext(httptest.NewRecorder(), req)

	controller := &Controller{
		manager: &fakeConfigManager{
			deleteStoreVisitFn: func(string) error {
				return fmtError(configmgmt.ErrStoreDisabled, "disabled")
			},
		},
	}

	_, err := controller.DeleteStoreVisitor(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
	assertHTTPCode(t, err, http.StatusNotFound)
}

func TestCreateStoreProxyIgnoresUnknownFields(t *testing.T) {
	var gotName string
	controller := &Controller{
		manager: &fakeConfigManager{
			createStoreProxyFn: func(cfg v1.ProxyConfigurer) (v1.ProxyConfigurer, error) {
				gotName = cfg.GetBaseConfig().Name
				return cfg, nil
			},
		},
	}

	body := []byte(`{"name":"raw-proxy","type":"tcp","unexpected":"value","tcp":{"localPort":10080,"unknownInBlock":"value"}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/store/proxies", bytes.NewReader(body))
	ctx := httppkg.NewContext(httptest.NewRecorder(), req)

	resp, err := controller.CreateStoreProxy(ctx)
	if err != nil {
		t.Fatalf("create store proxy: %v", err)
	}
	if gotName != "raw-proxy" {
		t.Fatalf("unexpected proxy name: %q", gotName)
	}

	payload, ok := resp.(model.ProxyDefinition)
	if !ok {
		t.Fatalf("unexpected response type: %T", resp)
	}
	if payload.Type != "tcp" || payload.TCP == nil {
		t.Fatalf("unexpected payload: %#v", payload)
	}
}

func TestCreateStoreVisitorIgnoresUnknownFields(t *testing.T) {
	var gotName string
	controller := &Controller{
		manager: &fakeConfigManager{
			createStoreVisitFn: func(cfg v1.VisitorConfigurer) (v1.VisitorConfigurer, error) {
				gotName = cfg.GetBaseConfig().Name
				return cfg, nil
			},
		},
	}

	body := []byte(`{
			"name":"raw-visitor","type":"xtcp","unexpected":"value",
			"xtcp":{"serverName":"server","bindPort":10081,"secretKey":"secret","unknownInBlock":"value"}
		}`)
	req := httptest.NewRequest(http.MethodPost, "/api/store/visitors", bytes.NewReader(body))
	ctx := httppkg.NewContext(httptest.NewRecorder(), req)

	resp, err := controller.CreateStoreVisitor(ctx)
	if err != nil {
		t.Fatalf("create store visitor: %v", err)
	}
	if gotName != "raw-visitor" {
		t.Fatalf("unexpected visitor name: %q", gotName)
	}

	payload, ok := resp.(model.VisitorDefinition)
	if !ok {
		t.Fatalf("unexpected response type: %T", resp)
	}
	if payload.Type != "xtcp" || payload.XTCP == nil {
		t.Fatalf("unexpected payload: %#v", payload)
	}
}

func TestCreateStoreProxyPluginUnknownFieldsAreIgnored(t *testing.T) {
	var gotPluginType string
	controller := &Controller{
		manager: &fakeConfigManager{
			createStoreProxyFn: func(cfg v1.ProxyConfigurer) (v1.ProxyConfigurer, error) {
				gotPluginType = cfg.GetBaseConfig().Plugin.Type
				return cfg, nil
			},
		},
	}

	body := []byte(`{"name":"plugin-proxy","type":"tcp","tcp":{"plugin":{"type":"http2https","localAddr":"127.0.0.1:8080","unknownInPlugin":"value"}}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/store/proxies", bytes.NewReader(body))
	ctx := httppkg.NewContext(httptest.NewRecorder(), req)

	resp, err := controller.CreateStoreProxy(ctx)
	if err != nil {
		t.Fatalf("create store proxy: %v", err)
	}
	if gotPluginType != "http2https" {
		t.Fatalf("unexpected plugin type: %q", gotPluginType)
	}
	payload, ok := resp.(model.ProxyDefinition)
	if !ok {
		t.Fatalf("unexpected response type: %T", resp)
	}
	if payload.TCP == nil {
		t.Fatalf("unexpected response payload: %#v", payload)
	}
	pluginType := payload.TCP.Plugin.Type

	if pluginType != "http2https" {
		t.Fatalf("unexpected plugin type in response payload: %q", pluginType)
	}
}

func TestCreateStoreVisitorPluginUnknownFieldsAreIgnored(t *testing.T) {
	var gotPluginType string
	controller := &Controller{
		manager: &fakeConfigManager{
			createStoreVisitFn: func(cfg v1.VisitorConfigurer) (v1.VisitorConfigurer, error) {
				gotPluginType = cfg.GetBaseConfig().Plugin.Type
				return cfg, nil
			},
		},
	}

	body := []byte(`{
			"name":"plugin-visitor","type":"stcp",
			"stcp":{"serverName":"server","bindPort":10081,"plugin":{"type":"virtual_net","destinationIP":"10.0.0.1","unknownInPlugin":"value"}}
		}`)
	req := httptest.NewRequest(http.MethodPost, "/api/store/visitors", bytes.NewReader(body))
	ctx := httppkg.NewContext(httptest.NewRecorder(), req)

	resp, err := controller.CreateStoreVisitor(ctx)
	if err != nil {
		t.Fatalf("create store visitor: %v", err)
	}
	if gotPluginType != "virtual_net" {
		t.Fatalf("unexpected plugin type: %q", gotPluginType)
	}
	payload, ok := resp.(model.VisitorDefinition)
	if !ok {
		t.Fatalf("unexpected response type: %T", resp)
	}
	if payload.STCP == nil {
		t.Fatalf("unexpected response payload: %#v", payload)
	}
	pluginType := payload.STCP.Plugin.Type

	if pluginType != "virtual_net" {
		t.Fatalf("unexpected plugin type in response payload: %q", pluginType)
	}
}

func TestUpdateStoreProxyRejectsMismatchedTypeBlock(t *testing.T) {
	controller := &Controller{manager: &fakeConfigManager{}}
	body := []byte(`{"name":"p1","type":"tcp","udp":{"localPort":10080}}`)
	req := httptest.NewRequest(http.MethodPut, "/api/store/proxies/p1", bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"name": "p1"})
	ctx := httppkg.NewContext(httptest.NewRecorder(), req)

	_, err := controller.UpdateStoreProxy(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
	assertHTTPCode(t, err, http.StatusBadRequest)
}

func TestUpdateStoreProxyRejectsNameMismatch(t *testing.T) {
	controller := &Controller{manager: &fakeConfigManager{}}
	body := []byte(`{"name":"p2","type":"tcp","tcp":{"localPort":10080}}`)
	req := httptest.NewRequest(http.MethodPut, "/api/store/proxies/p1", bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"name": "p1"})
	ctx := httppkg.NewContext(httptest.NewRecorder(), req)

	_, err := controller.UpdateStoreProxy(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
	assertHTTPCode(t, err, http.StatusBadRequest)
}

func TestListStoreProxiesReturnsSortedPayload(t *testing.T) {
	controller := &Controller{
		manager: &fakeConfigManager{
			listStoreProxiesFn: func() ([]v1.ProxyConfigurer, error) {
				b := newRawTCPProxyConfig("b")
				a := newRawTCPProxyConfig("a")
				return []v1.ProxyConfigurer{b, a}, nil
			},
		},
	}
	ctx := httppkg.NewContext(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/store/proxies", nil))

	resp, err := controller.ListStoreProxies(ctx)
	if err != nil {
		t.Fatalf("list store proxies: %v", err)
	}
	out, ok := resp.(model.ProxyListResp)
	if !ok {
		t.Fatalf("unexpected response type: %T", resp)
	}
	if len(out.Proxies) != 2 {
		t.Fatalf("unexpected proxy count: %d", len(out.Proxies))
	}
	if out.Proxies[0].Name != "a" || out.Proxies[1].Name != "b" {
		t.Fatalf("proxies are not sorted by name: %#v", out.Proxies)
	}
}

func fmtError(sentinel error, msg string) error {
	return fmt.Errorf("%w: %s", sentinel, msg)
}

func assertHTTPCode(t *testing.T, err error, expected int) {
	t.Helper()
	var httpErr *httppkg.Error
	if !errors.As(err, &httpErr) {
		t.Fatalf("unexpected error type: %T", err)
	}
	if httpErr.Code != expected {
		t.Fatalf("unexpected status code: got %d, want %d", httpErr.Code, expected)
	}
}

func TestUpdateStoreProxyReturnsTypedPayload(t *testing.T) {
	controller := &Controller{
		manager: &fakeConfigManager{
			updateStoreProxyFn: func(_ string, cfg v1.ProxyConfigurer) (v1.ProxyConfigurer, error) {
				return cfg, nil
			},
		},
	}

	body := map[string]any{
		"name": "shared-proxy",
		"type": "tcp",
		"tcp": map[string]any{
			"localPort":  10080,
			"remotePort": 7000,
		},
	}
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPut, "/api/store/proxies/shared-proxy", bytes.NewReader(data))
	req = mux.SetURLVars(req, map[string]string{"name": "shared-proxy"})
	ctx := httppkg.NewContext(httptest.NewRecorder(), req)

	resp, err := controller.UpdateStoreProxy(ctx)
	if err != nil {
		t.Fatalf("update store proxy: %v", err)
	}
	payload, ok := resp.(model.ProxyDefinition)
	if !ok {
		t.Fatalf("unexpected response type: %T", resp)
	}
	if payload.TCP == nil || payload.TCP.RemotePort != 7000 {
		t.Fatalf("unexpected response payload: %#v", payload)
	}
}

func TestGetProxyConfigFromManager(t *testing.T) {
	controller := &Controller{
		manager: &fakeConfigManager{
			getProxyConfigFn: func(name string) (v1.ProxyConfigurer, bool) {
				if name == "ssh" {
					cfg := &v1.TCPProxyConfig{
						ProxyBaseConfig: v1.ProxyBaseConfig{
							Name: "ssh",
							Type: "tcp",
							ProxyBackend: v1.ProxyBackend{
								LocalPort: 22,
							},
						},
					}
					return cfg, true
				}
				return nil, false
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/proxy/ssh/config", nil)
	req = mux.SetURLVars(req, map[string]string{"name": "ssh"})
	ctx := httppkg.NewContext(httptest.NewRecorder(), req)

	resp, err := controller.GetProxyConfig(ctx)
	if err != nil {
		t.Fatalf("get proxy config: %v", err)
	}
	payload, ok := resp.(model.ProxyDefinition)
	if !ok {
		t.Fatalf("unexpected response type: %T", resp)
	}
	if payload.Name != "ssh" || payload.Type != "tcp" || payload.TCP == nil {
		t.Fatalf("unexpected payload: %#v", payload)
	}
}

func TestGetProxyConfigNotFound(t *testing.T) {
	controller := &Controller{
		manager: &fakeConfigManager{
			getProxyConfigFn: func(name string) (v1.ProxyConfigurer, bool) {
				return nil, false
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/proxy/missing/config", nil)
	req = mux.SetURLVars(req, map[string]string{"name": "missing"})
	ctx := httppkg.NewContext(httptest.NewRecorder(), req)

	_, err := controller.GetProxyConfig(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
	assertHTTPCode(t, err, http.StatusNotFound)
}

func TestGetVisitorConfigFromManager(t *testing.T) {
	controller := &Controller{
		manager: &fakeConfigManager{
			getVisitorConfigFn: func(name string) (v1.VisitorConfigurer, bool) {
				if name == "my-stcp" {
					cfg := &v1.STCPVisitorConfig{
						VisitorBaseConfig: v1.VisitorBaseConfig{
							Name:       "my-stcp",
							Type:       "stcp",
							ServerName: "server1",
							BindPort:   9000,
						},
					}
					return cfg, true
				}
				return nil, false
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/visitor/my-stcp/config", nil)
	req = mux.SetURLVars(req, map[string]string{"name": "my-stcp"})
	ctx := httppkg.NewContext(httptest.NewRecorder(), req)

	resp, err := controller.GetVisitorConfig(ctx)
	if err != nil {
		t.Fatalf("get visitor config: %v", err)
	}
	payload, ok := resp.(model.VisitorDefinition)
	if !ok {
		t.Fatalf("unexpected response type: %T", resp)
	}
	if payload.Name != "my-stcp" || payload.Type != "stcp" || payload.STCP == nil {
		t.Fatalf("unexpected payload: %#v", payload)
	}
}

func TestGetVisitorConfigNotFound(t *testing.T) {
	controller := &Controller{
		manager: &fakeConfigManager{
			getVisitorConfigFn: func(name string) (v1.VisitorConfigurer, bool) {
				return nil, false
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/visitor/missing/config", nil)
	req = mux.SetURLVars(req, map[string]string{"name": "missing"})
	ctx := httppkg.NewContext(httptest.NewRecorder(), req)

	_, err := controller.GetVisitorConfig(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
	assertHTTPCode(t, err, http.StatusNotFound)
}
