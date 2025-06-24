package server

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/fatedier/frp/pkg/util/util"
	"github.com/fatedier/frp/pkg/util/xlog"
)

type CompositeOperationPluginGateway struct {
	router *PluginOpsRouter
}

func NewManager() *CompositeOperationPluginGateway {
	return &CompositeOperationPluginGateway{
		router: NewPluginOpsRouter(),
	}
}

type PluginOpsRouter struct {
	operations map[string][]Plugin
}

func NewPluginOpsRouter() *PluginOpsRouter {
	return &PluginOpsRouter{
		operations: make(map[string][]Plugin),
	}
}

func (r *PluginOpsRouter) AddPlugin(op string, p Plugin) {
	r.operations[op] = append(r.operations[op], p)
}

func (m *CompositeOperationPluginGateway) Register(p Plugin) {
	for _, op := range []string{OpLogin, OpNewProxy, OpCloseProxy, OpPing, OpNewWorkConn, OpNewUserConn} {
		if p.IsSupport(op) {
			m.router.AddPlugin(op, p)
		}
	}
}

func buildCtx() (context.Context, string) {
	reqid, _ := util.RandID()
	xl := xlog.New().AppendPrefix("reqid: " + reqid)
	ctx := xlog.NewContext(context.Background(), xl)
	ctx = NewReqidContext(ctx, reqid)
	return ctx, reqid
}

func (m *CompositeOperationPluginGateway) doRequest(op string, input any) (any, error) {
	ctx, _ := buildCtx()
	plugins := m.router.operations[op]
	if len(plugins) == 0 {
		return input, nil
	}

	response := &Response{
		Reject:   false,
		Unchange: true,
	}

	var retContent any
	var err error
	for _, p := range plugins {
		response, retContent, err = p.Handle(ctx, op, input)
		if err != nil {
			return nil, fmt.Errorf("plugin [%s] failed: %v", p.Name(), err)
		}
		if response.Reject {
			return nil, fmt.Errorf("%s", response.RejectReason)
		}
		if !response.Unchange {
			input = retContent
		}
	}
	return input, nil
}

func (m *CompositeOperationPluginGateway) Login(c *LoginContent) (*LoginContent, error) {
	out, err := m.doRequest(OpLogin, *c)
	if err != nil {
		return nil, err
	}
	return out.(*LoginContent), nil
}

func (m *CompositeOperationPluginGateway) NewProxy(c *NewProxyContent) (*NewProxyContent, error) {
	out, err := m.doRequest(OpNewProxy, *c)
	if err != nil {
		return nil, err
	}
	return out.(*NewProxyContent), nil
}

func (m *CompositeOperationPluginGateway) Ping(c *PingContent) (*PingContent, error) {
	out, err := m.doRequest(OpPing, *c)
	if err != nil {
		return nil, err
	}
	return out.(*PingContent), nil
}

func (m *CompositeOperationPluginGateway) NewWorkConn(c *NewWorkConnContent) (*NewWorkConnContent, error) {
	out, err := m.doRequest(OpNewWorkConn, *c)
	if err != nil {
		return nil, err
	}
	return out.(*NewWorkConnContent), nil
}

func (m *CompositeOperationPluginGateway) NewUserConn(c *NewUserConnContent) (*NewUserConnContent, error) {
	out, err := m.doRequest(OpNewUserConn, *c)
	if err != nil {
		return nil, err
	}
	return out.(*NewUserConnContent), nil
}

func (m *CompositeOperationPluginGateway) CloseProxy(c *CloseProxyContent) error {
	ctx, _ := buildCtx()
	errs := []string{}
	for _, p := range m.router.operations[OpCloseProxy] {
		_, _, err := p.Handle(ctx, OpCloseProxy, *c)
		if err != nil {
			errs = append(errs, fmt.Sprintf("[%s]: %v", p.Name(), err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("plugin CloseProxy errors: %s", strings.Join(errs, "; "))
	}
	return nil
}
