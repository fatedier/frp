package client

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/fatedier/frp/client/configmgmt"
	"github.com/fatedier/frp/client/proxy"
	"github.com/fatedier/frp/pkg/config"
	"github.com/fatedier/frp/pkg/config/source"
	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/config/v1/validation"
	"github.com/fatedier/frp/pkg/util/log"
)

type serviceConfigManager struct {
	svr *Service
}

func newServiceConfigManager(svr *Service) configmgmt.ConfigManager {
	return &serviceConfigManager{svr: svr}
}

func (m *serviceConfigManager) ReloadFromFile(strict bool) error {
	if m.svr.configFilePath == "" {
		return fmt.Errorf("%w: frpc has no config file path", configmgmt.ErrInvalidArgument)
	}

	result, err := config.LoadClientConfigResult(m.svr.configFilePath, strict)
	if err != nil {
		return fmt.Errorf("%w: %v", configmgmt.ErrInvalidArgument, err)
	}

	proxyCfgsForValidation, visitorCfgsForValidation := config.FilterClientConfigurers(
		result.Common,
		result.Proxies,
		result.Visitors,
	)
	proxyCfgsForValidation = config.CompleteProxyConfigurers(proxyCfgsForValidation)
	visitorCfgsForValidation = config.CompleteVisitorConfigurers(visitorCfgsForValidation)

	if _, err := validation.ValidateAllClientConfig(result.Common, proxyCfgsForValidation, visitorCfgsForValidation, m.svr.unsafeFeatures); err != nil {
		return fmt.Errorf("%w: %v", configmgmt.ErrInvalidArgument, err)
	}

	if err := m.svr.UpdateConfigSource(result.Common, result.Proxies, result.Visitors); err != nil {
		return fmt.Errorf("%w: %v", configmgmt.ErrApplyConfig, err)
	}

	log.Infof("success reload conf")
	return nil
}

func (m *serviceConfigManager) ReadConfigFile() (string, error) {
	if m.svr.configFilePath == "" {
		return "", fmt.Errorf("%w: frpc has no config file path", configmgmt.ErrInvalidArgument)
	}

	content, err := os.ReadFile(m.svr.configFilePath)
	if err != nil {
		return "", fmt.Errorf("%w: %v", configmgmt.ErrInvalidArgument, err)
	}
	return string(content), nil
}

func (m *serviceConfigManager) WriteConfigFile(content []byte) error {
	if len(content) == 0 {
		return fmt.Errorf("%w: body can't be empty", configmgmt.ErrInvalidArgument)
	}

	if err := os.WriteFile(m.svr.configFilePath, content, 0o600); err != nil {
		return err
	}
	return nil
}

func (m *serviceConfigManager) GetProxyStatus() []*proxy.WorkingStatus {
	return m.svr.getAllProxyStatus()
}

func (m *serviceConfigManager) IsStoreProxyEnabled(name string) bool {
	if name == "" {
		return false
	}

	m.svr.reloadMu.Lock()
	storeSource := m.svr.storeSource
	m.svr.reloadMu.Unlock()

	if storeSource == nil {
		return false
	}

	cfg := storeSource.GetProxy(name)
	if cfg == nil {
		return false
	}
	enabled := cfg.GetBaseConfig().Enabled
	return enabled == nil || *enabled
}

func (m *serviceConfigManager) StoreEnabled() bool {
	m.svr.reloadMu.Lock()
	storeSource := m.svr.storeSource
	m.svr.reloadMu.Unlock()
	return storeSource != nil
}

func (m *serviceConfigManager) ListStoreProxies() ([]v1.ProxyConfigurer, error) {
	storeSource, err := m.storeSourceOrError()
	if err != nil {
		return nil, err
	}
	return storeSource.GetAllProxies()
}

func (m *serviceConfigManager) GetStoreProxy(name string) (v1.ProxyConfigurer, error) {
	if name == "" {
		return nil, fmt.Errorf("%w: proxy name is required", configmgmt.ErrInvalidArgument)
	}

	storeSource, err := m.storeSourceOrError()
	if err != nil {
		return nil, err
	}

	cfg := storeSource.GetProxy(name)
	if cfg == nil {
		return nil, fmt.Errorf("%w: proxy %q", configmgmt.ErrNotFound, name)
	}
	return cfg, nil
}

func (m *serviceConfigManager) CreateStoreProxy(cfg v1.ProxyConfigurer) error {
	if err := m.validateStoreProxyConfigurer(cfg); err != nil {
		return fmt.Errorf("%w: validation error: %v", configmgmt.ErrInvalidArgument, err)
	}

	if err := m.withStoreMutationAndReload(func(storeSource *source.StoreSource) error {
		if err := storeSource.AddProxy(cfg); err != nil {
			if errors.Is(err, source.ErrAlreadyExists) {
				return fmt.Errorf("%w: %v", configmgmt.ErrConflict, err)
			}
			return err
		}
		return nil
	}); err != nil {
		return err
	}

	log.Infof("store: created proxy %q", cfg.GetBaseConfig().Name)
	return nil
}

func (m *serviceConfigManager) UpdateStoreProxy(name string, cfg v1.ProxyConfigurer) error {
	if name == "" {
		return fmt.Errorf("%w: proxy name is required", configmgmt.ErrInvalidArgument)
	}
	if cfg == nil {
		return fmt.Errorf("%w: invalid proxy config: type is required", configmgmt.ErrInvalidArgument)
	}
	bodyName := cfg.GetBaseConfig().Name
	if bodyName != name {
		return fmt.Errorf("%w: proxy name in URL must match name in body", configmgmt.ErrInvalidArgument)
	}
	if err := m.validateStoreProxyConfigurer(cfg); err != nil {
		return fmt.Errorf("%w: validation error: %v", configmgmt.ErrInvalidArgument, err)
	}

	if err := m.withStoreMutationAndReload(func(storeSource *source.StoreSource) error {
		if err := storeSource.UpdateProxy(cfg); err != nil {
			if errors.Is(err, source.ErrNotFound) {
				return fmt.Errorf("%w: %v", configmgmt.ErrNotFound, err)
			}
			return err
		}
		return nil
	}); err != nil {
		return err
	}

	log.Infof("store: updated proxy %q", name)
	return nil
}

func (m *serviceConfigManager) DeleteStoreProxy(name string) error {
	if name == "" {
		return fmt.Errorf("%w: proxy name is required", configmgmt.ErrInvalidArgument)
	}

	if err := m.withStoreMutationAndReload(func(storeSource *source.StoreSource) error {
		if err := storeSource.RemoveProxy(name); err != nil {
			if errors.Is(err, source.ErrNotFound) {
				return fmt.Errorf("%w: %v", configmgmt.ErrNotFound, err)
			}
			return err
		}
		return nil
	}); err != nil {
		return err
	}

	log.Infof("store: deleted proxy %q", name)
	return nil
}

func (m *serviceConfigManager) ListStoreVisitors() ([]v1.VisitorConfigurer, error) {
	storeSource, err := m.storeSourceOrError()
	if err != nil {
		return nil, err
	}
	return storeSource.GetAllVisitors()
}

func (m *serviceConfigManager) GetStoreVisitor(name string) (v1.VisitorConfigurer, error) {
	if name == "" {
		return nil, fmt.Errorf("%w: visitor name is required", configmgmt.ErrInvalidArgument)
	}

	storeSource, err := m.storeSourceOrError()
	if err != nil {
		return nil, err
	}

	cfg := storeSource.GetVisitor(name)
	if cfg == nil {
		return nil, fmt.Errorf("%w: visitor %q", configmgmt.ErrNotFound, name)
	}
	return cfg, nil
}

func (m *serviceConfigManager) CreateStoreVisitor(cfg v1.VisitorConfigurer) error {
	if err := m.validateStoreVisitorConfigurer(cfg); err != nil {
		return fmt.Errorf("%w: validation error: %v", configmgmt.ErrInvalidArgument, err)
	}

	if err := m.withStoreMutationAndReload(func(storeSource *source.StoreSource) error {
		if err := storeSource.AddVisitor(cfg); err != nil {
			if errors.Is(err, source.ErrAlreadyExists) {
				return fmt.Errorf("%w: %v", configmgmt.ErrConflict, err)
			}
			return err
		}
		return nil
	}); err != nil {
		return err
	}

	log.Infof("store: created visitor %q", cfg.GetBaseConfig().Name)
	return nil
}

func (m *serviceConfigManager) UpdateStoreVisitor(name string, cfg v1.VisitorConfigurer) error {
	if name == "" {
		return fmt.Errorf("%w: visitor name is required", configmgmt.ErrInvalidArgument)
	}
	if cfg == nil {
		return fmt.Errorf("%w: invalid visitor config: type is required", configmgmt.ErrInvalidArgument)
	}
	bodyName := cfg.GetBaseConfig().Name
	if bodyName != name {
		return fmt.Errorf("%w: visitor name in URL must match name in body", configmgmt.ErrInvalidArgument)
	}
	if err := m.validateStoreVisitorConfigurer(cfg); err != nil {
		return fmt.Errorf("%w: validation error: %v", configmgmt.ErrInvalidArgument, err)
	}

	if err := m.withStoreMutationAndReload(func(storeSource *source.StoreSource) error {
		if err := storeSource.UpdateVisitor(cfg); err != nil {
			if errors.Is(err, source.ErrNotFound) {
				return fmt.Errorf("%w: %v", configmgmt.ErrNotFound, err)
			}
			return err
		}
		return nil
	}); err != nil {
		return err
	}

	log.Infof("store: updated visitor %q", name)
	return nil
}

func (m *serviceConfigManager) DeleteStoreVisitor(name string) error {
	if name == "" {
		return fmt.Errorf("%w: visitor name is required", configmgmt.ErrInvalidArgument)
	}

	if err := m.withStoreMutationAndReload(func(storeSource *source.StoreSource) error {
		if err := storeSource.RemoveVisitor(name); err != nil {
			if errors.Is(err, source.ErrNotFound) {
				return fmt.Errorf("%w: %v", configmgmt.ErrNotFound, err)
			}
			return err
		}
		return nil
	}); err != nil {
		return err
	}

	log.Infof("store: deleted visitor %q", name)
	return nil
}

func (m *serviceConfigManager) GracefulClose(d time.Duration) {
	m.svr.GracefulClose(d)
}

func (m *serviceConfigManager) storeSourceOrError() (*source.StoreSource, error) {
	m.svr.reloadMu.Lock()
	storeSource := m.svr.storeSource
	m.svr.reloadMu.Unlock()

	if storeSource == nil {
		return nil, fmt.Errorf("%w: store API is disabled", configmgmt.ErrStoreDisabled)
	}
	return storeSource, nil
}

func (m *serviceConfigManager) withStoreMutationAndReload(
	fn func(storeSource *source.StoreSource) error,
) error {
	m.svr.reloadMu.Lock()
	defer m.svr.reloadMu.Unlock()

	storeSource := m.svr.storeSource
	if storeSource == nil {
		return fmt.Errorf("%w: store API is disabled", configmgmt.ErrStoreDisabled)
	}

	if err := fn(storeSource); err != nil {
		return err
	}

	if err := m.svr.reloadConfigFromSourcesLocked(); err != nil {
		return fmt.Errorf("%w: failed to apply config: %v", configmgmt.ErrApplyConfig, err)
	}
	return nil
}

func (m *serviceConfigManager) validateStoreProxyConfigurer(cfg v1.ProxyConfigurer) error {
	if cfg == nil {
		return fmt.Errorf("invalid proxy config")
	}
	runtimeCfg := cfg.Clone()
	if runtimeCfg == nil {
		return fmt.Errorf("invalid proxy config")
	}
	runtimeCfg.Complete()
	return validation.ValidateProxyConfigurerForClient(runtimeCfg)
}

func (m *serviceConfigManager) validateStoreVisitorConfigurer(cfg v1.VisitorConfigurer) error {
	if cfg == nil {
		return fmt.Errorf("invalid visitor config")
	}
	runtimeCfg := cfg.Clone()
	if runtimeCfg == nil {
		return fmt.Errorf("invalid visitor config")
	}
	runtimeCfg.Complete()
	return validation.ValidateVisitorConfigurer(runtimeCfg)
}
