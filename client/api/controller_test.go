package api

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

	listStoreProxiesFn  func() ([]v1.ProxyConfigurer, error)
	getStoreProxyFn     func(name string) (v1.ProxyConfigurer, error)
	createStoreProxyFn  func(cfg v1.ProxyConfigurer) error
	updateStoreProxyFn  func(name string, cfg v1.ProxyConfigurer) error
	deleteStoreProxyFn  func(name string) error
	listStoreVisitorsFn func() ([]v1.VisitorConfigurer, error)
	getStoreVisitorFn   func(name string) (v1.VisitorConfigurer, error)
	createStoreVisitFn  func(cfg v1.VisitorConfigurer) error
	updateStoreVisitFn  func(name string, cfg v1.VisitorConfigurer) error
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

func (m *fakeConfigManager) CreateStoreProxy(cfg v1.ProxyConfigurer) error {
	if m.createStoreProxyFn != nil {
		return m.createStoreProxyFn(cfg)
	}
	return nil
}

func (m *fakeConfigManager) UpdateStoreProxy(name string, cfg v1.ProxyConfigurer) error {
	if m.updateStoreProxyFn != nil {
		return m.updateStoreProxyFn(name, cfg)
	}
	return nil
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

func (m *fakeConfigManager) CreateStoreVisitor(cfg v1.VisitorConfigurer) error {
	if m.createStoreVisitFn != nil {
		return m.createStoreVisitFn(cfg)
	}
	return nil
}

func (m *fakeConfigManager) UpdateStoreVisitor(name string, cfg v1.VisitorConfigurer) error {
	if m.updateStoreVisitFn != nil {
		return m.updateStoreVisitFn(name, cfg)
	}
	return nil
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

func setDisallowUnknownFieldsForTest(t *testing.T, value bool) func() {
	t.Helper()
	v1.DisallowUnknownFieldsMu.Lock()
	prev := v1.DisallowUnknownFields
	v1.DisallowUnknownFields = value
	v1.DisallowUnknownFieldsMu.Unlock()
	return func() {
		v1.DisallowUnknownFieldsMu.Lock()
		v1.DisallowUnknownFields = prev
		v1.DisallowUnknownFieldsMu.Unlock()
	}
}

func getDisallowUnknownFieldsForTest() bool {
	v1.DisallowUnknownFieldsMu.Lock()
	defer v1.DisallowUnknownFieldsMu.Unlock()
	return v1.DisallowUnknownFields
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

func newRawXTCPVisitorConfig(name string) *v1.XTCPVisitorConfig {
	return &v1.XTCPVisitorConfig{
		VisitorBaseConfig: v1.VisitorBaseConfig{
			Name:       name,
			Type:       "xtcp",
			ServerName: "server",
			BindPort:   10081,
			SecretKey:  "secret",
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
			body, err := json.Marshal(newRawTCPProxyConfig("shared-proxy"))
			if err != nil {
				t.Fatalf("marshal body: %v", err)
			}

			req := httptest.NewRequest(http.MethodPut, "/api/store/proxies/shared-proxy", bytes.NewReader(body))
			req = mux.SetURLVars(req, map[string]string{"name": "shared-proxy"})
			ctx := httppkg.NewContext(httptest.NewRecorder(), req)

			controller := &Controller{
				manager: &fakeConfigManager{
					updateStoreProxyFn: func(_ string, _ v1.ProxyConfigurer) error { return tc.err },
				},
			}

			_, err = controller.UpdateStoreProxy(ctx)
			if err == nil {
				t.Fatal("expected error")
			}
			assertHTTPCode(t, err, tc.expectedCode)
		})
	}
}

func TestStoreVisitorErrorMapping(t *testing.T) {
	body, err := json.Marshal(newRawXTCPVisitorConfig("shared-visitor"))
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}

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

	_, err = controller.DeleteStoreVisitor(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
	assertHTTPCode(t, err, http.StatusNotFound)
}

func TestCreateStoreProxy_UnknownFieldsNotAffectedByAmbientStrictness(t *testing.T) {
	restore := setDisallowUnknownFieldsForTest(t, true)
	t.Cleanup(restore)

	var gotName string
	controller := &Controller{
		manager: &fakeConfigManager{
			createStoreProxyFn: func(cfg v1.ProxyConfigurer) error {
				gotName = cfg.GetBaseConfig().Name
				return nil
			},
		},
	}

	body := []byte(`{"name":"raw-proxy","type":"tcp","localPort":10080,"unexpected":"value"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/store/proxies", bytes.NewReader(body))
	ctx := httppkg.NewContext(httptest.NewRecorder(), req)

	_, err := controller.CreateStoreProxy(ctx)
	if err != nil {
		t.Fatalf("create store proxy: %v", err)
	}
	if gotName != "raw-proxy" {
		t.Fatalf("unexpected proxy name: %q", gotName)
	}
	if !getDisallowUnknownFieldsForTest() {
		t.Fatal("global strictness flag was not restored")
	}
}

func TestCreateStoreVisitor_UnknownFieldsNotAffectedByAmbientStrictness(t *testing.T) {
	restore := setDisallowUnknownFieldsForTest(t, true)
	t.Cleanup(restore)

	var gotName string
	controller := &Controller{
		manager: &fakeConfigManager{
			createStoreVisitFn: func(cfg v1.VisitorConfigurer) error {
				gotName = cfg.GetBaseConfig().Name
				return nil
			},
		},
	}

	body := []byte(`{"name":"raw-visitor","type":"xtcp","serverName":"server","bindPort":10081,"secretKey":"secret","unexpected":"value"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/store/visitors", bytes.NewReader(body))
	ctx := httppkg.NewContext(httptest.NewRecorder(), req)

	_, err := controller.CreateStoreVisitor(ctx)
	if err != nil {
		t.Fatalf("create store visitor: %v", err)
	}
	if gotName != "raw-visitor" {
		t.Fatalf("unexpected visitor name: %q", gotName)
	}
	if !getDisallowUnknownFieldsForTest() {
		t.Fatal("global strictness flag was not restored")
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
