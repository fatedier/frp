// Copyright 2017 fatedier, fatedier@gmail.com
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

package proxy

import (
	"context"
	"fmt"
	"io"
	"net"
	"reflect"
	"strconv"
	"sync"
	"time"

	libio "github.com/fatedier/golib/io"
	"golang.org/x/time/rate"

	"github.com/fatedier/frp/pkg/config/types"
	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/msg"
	plugin "github.com/fatedier/frp/pkg/plugin/server"
	"github.com/fatedier/frp/pkg/util/limit"
	netpkg "github.com/fatedier/frp/pkg/util/net"
	"github.com/fatedier/frp/pkg/util/xlog"
	"github.com/fatedier/frp/server/controller"
	"github.com/fatedier/frp/server/metrics"
)

var proxyFactoryRegistry = map[reflect.Type]func(*BaseProxy) Proxy{}

func RegisterProxyFactory(proxyConfType reflect.Type, factory func(*BaseProxy) Proxy) {
	proxyFactoryRegistry[proxyConfType] = factory
}

type GetWorkConnFn func() (net.Conn, error)

type Proxy interface {
	Context() context.Context
	Run() (remoteAddr string, err error)
	GetName() string
	GetConfigurer() v1.ProxyConfigurer
	GetWorkConnFromPool(src, dst net.Addr) (workConn net.Conn, err error)
	GetUsedPortsNum() int
	GetResourceController() *controller.ResourceController
	GetUserInfo() plugin.UserInfo
	GetLimiter() *rate.Limiter
	GetLoginMsg() *msg.Login
	Close()
}

type BaseProxy struct {
	name          string
	rc            *controller.ResourceController
	listeners     []net.Listener
	usedPortsNum  int
	poolCount     int
	getWorkConnFn GetWorkConnFn
	serverCfg     *v1.ServerConfig
	limiter       *rate.Limiter
	userInfo      plugin.UserInfo
	loginMsg      *msg.Login
	configurer    v1.ProxyConfigurer

	mu  sync.RWMutex
	xl  *xlog.Logger
	ctx context.Context
}

func (pxy *BaseProxy) GetName() string {
	return pxy.name
}

func (pxy *BaseProxy) Context() context.Context {
	return pxy.ctx
}

func (pxy *BaseProxy) GetUsedPortsNum() int {
	return pxy.usedPortsNum
}

func (pxy *BaseProxy) GetResourceController() *controller.ResourceController {
	return pxy.rc
}

func (pxy *BaseProxy) GetUserInfo() plugin.UserInfo {
	return pxy.userInfo
}

func (pxy *BaseProxy) GetLoginMsg() *msg.Login {
	return pxy.loginMsg
}

func (pxy *BaseProxy) GetLimiter() *rate.Limiter {
	return pxy.limiter
}

func (pxy *BaseProxy) GetConfigurer() v1.ProxyConfigurer {
	return pxy.configurer
}

func (pxy *BaseProxy) Close() {
	xl := xlog.FromContextSafe(pxy.ctx)
	xl.Infof("proxy closing")
	for _, l := range pxy.listeners {
		l.Close()
	}
}

// GetWorkConnFromPool try to get a new work connections from pool
// for quickly response, we immediately send the StartWorkConn message to frpc after take out one from pool
func (pxy *BaseProxy) GetWorkConnFromPool(src, dst net.Addr) (workConn net.Conn, err error) {
	xl := xlog.FromContextSafe(pxy.ctx)
	// try all connections from the pool
	for i := 0; i < pxy.poolCount+1; i++ {
		if workConn, err = pxy.getWorkConnFn(); err != nil {
			xl.Warnf("failed to get work connection: %v", err)
			return
		}
		xl.Debugf("get a new work connection: [%s]", workConn.RemoteAddr().String())
		xl.Spawn().AppendPrefix(pxy.GetName())
		workConn = netpkg.NewContextConn(pxy.ctx, workConn)

		var (
			srcAddr    string
			dstAddr    string
			srcPortStr string
			dstPortStr string
			srcPort    uint64
			dstPort    uint64
		)

		if src != nil {
			srcAddr, srcPortStr, _ = net.SplitHostPort(src.String())
			srcPort, _ = strconv.ParseUint(srcPortStr, 10, 16)
		}
		if dst != nil {
			dstAddr, dstPortStr, _ = net.SplitHostPort(dst.String())
			dstPort, _ = strconv.ParseUint(dstPortStr, 10, 16)
		}
		err := msg.WriteMsg(workConn, &msg.StartWorkConn{
			ProxyName: pxy.GetName(),
			SrcAddr:   srcAddr,
			SrcPort:   uint16(srcPort),
			DstAddr:   dstAddr,
			DstPort:   uint16(dstPort),
			Error:     "",
		})
		if err != nil {
			xl.Warnf("failed to send message to work connection from pool: %v, times: %d", err, i)
			workConn.Close()
		} else {
			break
		}
	}

	if err != nil {
		xl.Errorf("try to get work connection failed in the end")
		return
	}
	return
}

// startCommonTCPListenersHandler start a goroutine handler for each listener.
func (pxy *BaseProxy) startCommonTCPListenersHandler() {
	xl := xlog.FromContextSafe(pxy.ctx)
	for _, listener := range pxy.listeners {
		go func(l net.Listener) {
			var tempDelay time.Duration // how long to sleep on accept failure

			for {
				// block
				// if listener is closed, err returned
				c, err := l.Accept()
				if err != nil {
					if err, ok := err.(interface{ Temporary() bool }); ok && err.Temporary() {
						if tempDelay == 0 {
							tempDelay = 5 * time.Millisecond
						} else {
							tempDelay *= 2
						}
						if maxTime := 1 * time.Second; tempDelay > maxTime {
							tempDelay = maxTime
						}
						xl.Infof("met temporary error: %s, sleep for %s ...", err, tempDelay)
						time.Sleep(tempDelay)
						continue
					}

					xl.Warnf("listener is closed: %s", err)
					return
				}
				xl.Infof("get a user connection [%s]", c.RemoteAddr().String())
				go pxy.handleUserTCPConnection(c)
			}
		}(listener)
	}
}

// HandleUserTCPConnection is used for incoming user TCP connections.
func (pxy *BaseProxy) handleUserTCPConnection(userConn net.Conn) {
	xl := xlog.FromContextSafe(pxy.Context())
	defer userConn.Close()

	serverCfg := pxy.serverCfg
	cfg := pxy.configurer.GetBaseConfig()
	// server plugin hook
	rc := pxy.GetResourceController()
	content := &plugin.NewUserConnContent{
		User:       pxy.GetUserInfo(),
		ProxyName:  pxy.GetName(),
		ProxyType:  cfg.Type,
		RemoteAddr: userConn.RemoteAddr().String(),
	}
	_, err := rc.PluginManager.NewUserConn(content)
	if err != nil {
		xl.Warnf("the user conn [%s] was rejected, err:%v", content.RemoteAddr, err)
		return
	}

	// try all connections from the pool
	workConn, err := pxy.GetWorkConnFromPool(userConn.RemoteAddr(), userConn.LocalAddr())
	if err != nil {
		return
	}
	defer workConn.Close()

	var local io.ReadWriteCloser = workConn
	xl.Tracef("handler user tcp connection, use_encryption: %t, use_compression: %t",
		cfg.Transport.UseEncryption, cfg.Transport.UseCompression)
	if cfg.Transport.UseEncryption {
		key := []byte(serverCfg.Auth.Token)
		if serverCfg.Auth.Method == v1.AuthMethodJWT {
			key = []byte(pxy.loginMsg.PrivilegeKey)
		}
		local, err = libio.WithEncryption(local, key)
		if err != nil {
			xl.Errorf("create encryption stream error: %v", err)
			return
		}
	}
	if cfg.Transport.UseCompression {
		var recycleFn func()
		local, recycleFn = libio.WithCompressionFromPool(local)
		defer recycleFn()
	}

	if pxy.GetLimiter() != nil {
		local = libio.WrapReadWriteCloser(limit.NewReader(local, pxy.GetLimiter()), limit.NewWriter(local, pxy.GetLimiter()), func() error {
			return local.Close()
		})
	}

	xl.Debugf("join connections, workConn(l[%s] r[%s]) userConn(l[%s] r[%s])", workConn.LocalAddr().String(),
		workConn.RemoteAddr().String(), userConn.LocalAddr().String(), userConn.RemoteAddr().String())

	name := pxy.GetName()
	proxyType := cfg.Type
	metrics.Server.OpenConnection(name, proxyType)
	inCount, outCount, _ := libio.Join(local, userConn)
	metrics.Server.CloseConnection(name, proxyType)
	metrics.Server.AddTrafficIn(name, proxyType, inCount)
	metrics.Server.AddTrafficOut(name, proxyType, outCount)
	xl.Debugf("join connections closed")
}

type Options struct {
	UserInfo           plugin.UserInfo
	LoginMsg           *msg.Login
	PoolCount          int
	ResourceController *controller.ResourceController
	GetWorkConnFn      GetWorkConnFn
	Configurer         v1.ProxyConfigurer
	ServerCfg          *v1.ServerConfig
}

func NewProxy(ctx context.Context, options *Options) (pxy Proxy, err error) {
	configurer := options.Configurer
	xl := xlog.FromContextSafe(ctx).Spawn().AppendPrefix(configurer.GetBaseConfig().Name)

	var limiter *rate.Limiter
	limitBytes := configurer.GetBaseConfig().Transport.BandwidthLimit.Bytes()
	if limitBytes > 0 && configurer.GetBaseConfig().Transport.BandwidthLimitMode == types.BandwidthLimitModeServer {
		limiter = rate.NewLimiter(rate.Limit(float64(limitBytes)), int(limitBytes))
	}

	basePxy := BaseProxy{
		name:          configurer.GetBaseConfig().Name,
		rc:            options.ResourceController,
		listeners:     make([]net.Listener, 0),
		poolCount:     options.PoolCount,
		getWorkConnFn: options.GetWorkConnFn,
		serverCfg:     options.ServerCfg,
		limiter:       limiter,
		xl:            xl,
		ctx:           xlog.NewContext(ctx, xl),
		userInfo:      options.UserInfo,
		loginMsg:      options.LoginMsg,
		configurer:    configurer,
	}

	factory := proxyFactoryRegistry[reflect.TypeOf(configurer)]
	if factory == nil {
		return pxy, fmt.Errorf("proxy type not support")
	}
	pxy = factory(&basePxy)
	if pxy == nil {
		return nil, fmt.Errorf("proxy not created")
	}
	return pxy, nil
}

type Manager struct {
	// proxies indexed by proxy name
	pxys map[string]Proxy

	mu sync.RWMutex
}

func NewManager() *Manager {
	return &Manager{
		pxys: make(map[string]Proxy),
	}
}

func (pm *Manager) Add(name string, pxy Proxy) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	if _, ok := pm.pxys[name]; ok {
		return fmt.Errorf("proxy name [%s] is already in use", name)
	}

	pm.pxys[name] = pxy
	return nil
}

func (pm *Manager) Exist(name string) bool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	_, ok := pm.pxys[name]
	return ok
}

func (pm *Manager) Del(name string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	delete(pm.pxys, name)
}

func (pm *Manager) GetByName(name string) (pxy Proxy, ok bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	pxy, ok = pm.pxys[name]
	return
}
