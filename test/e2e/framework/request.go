package framework

import (
	"github.com/fatedier/frp/test/e2e/framework/consts"
	"github.com/fatedier/frp/test/e2e/pkg/request"
)

func SetRequestProtocol(protocol string) func(*request.Request) {
	return func(r *request.Request) {
		r.Protocol(protocol)
	}
}

func SetRequestPort(port int) func(*request.Request) {
	return func(r *request.Request) {
		r.Port(port)
	}
}

// NewRequest return a default TCP request with default timeout and content.
func NewRequest() *request.Request {
	return request.New().
		Timeout(consts.DefaultTimeout).
		Body([]byte(consts.TestString))
}

func ExpectResponse(req *request.Request, expectResp []byte, explain ...interface{}) {
	ret, err := req.Do()
	ExpectNoError(err, explain...)
	ExpectEqualValues(expectResp, ret, explain...)
}

func ExpectResponseError(req *request.Request, explain ...interface{}) {
	_, err := req.Do()
	ExpectError(err, explain...)
}

type RequestExpect struct {
	req *request.Request

	f           *Framework
	expectResp  []byte
	expectError bool
	explain     []interface{}
}

func NewRequestExpect(f *Framework) *RequestExpect {
	return &RequestExpect{
		req:         NewRequest(),
		f:           f,
		expectResp:  []byte(consts.TestString),
		expectError: false,
		explain:     make([]interface{}, 0),
	}
}

func (e *RequestExpect) RequestModify(f func(r *request.Request)) *RequestExpect {
	f(e.req)
	return e
}

func (e *RequestExpect) PortName(name string) *RequestExpect {
	if e.f != nil {
		e.req.Port(e.f.PortByName(name))
	}
	return e
}

func (e *RequestExpect) ExpectResp(resp []byte) *RequestExpect {
	e.expectResp = resp
	return e
}

func (e *RequestExpect) ExpectError(expectErr bool) *RequestExpect {
	e.expectError = expectErr
	return e
}

func (e *RequestExpect) Explain(explain ...interface{}) *RequestExpect {
	e.explain = explain
	return e
}

func (e *RequestExpect) Ensure() {
	if e.expectError {
		ExpectResponseError(e.req, e.explain...)
	} else {
		ExpectResponse(e.req, e.expectResp, e.explain...)
	}
}
