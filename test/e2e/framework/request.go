package framework

import (
	"bytes"
	"net/http"

	flog "github.com/fatedier/frp/pkg/util/log"
	"github.com/fatedier/frp/test/e2e/framework/consts"
	"github.com/fatedier/frp/test/e2e/pkg/request"
)

func SpecifiedHTTPBodyHandler(body []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write(body)
	}
}

func ExpectResponseCode(code int) EnsureFunc {
	return func(resp *request.Response) bool {
		if resp.Code == code {
			return true
		}
		flog.Warnf("Expect code %d, but got %d", code, resp.Code)
		return false
	}
}

// NewRequest return a default request with default timeout and content.
func NewRequest() *request.Request {
	return request.New().
		Timeout(consts.DefaultTimeout).
		Body([]byte(consts.TestString))
}

func NewHTTPRequest() *request.Request {
	return request.New().HTTP().HTTPParams("GET", "", "/", nil)
}

type RequestExpect struct {
	req *request.Request

	f           *Framework
	expectResp  []byte
	expectError bool
	explain     []any
}

func NewRequestExpect(f *Framework) *RequestExpect {
	return &RequestExpect{
		req:         NewRequest(),
		f:           f,
		expectResp:  []byte(consts.TestString),
		expectError: false,
		explain:     make([]any, 0),
	}
}

func (e *RequestExpect) Request(req *request.Request) *RequestExpect {
	e.req = req
	return e
}

func (e *RequestExpect) RequestModify(f func(r *request.Request)) *RequestExpect {
	f(e.req)
	return e
}

func (e *RequestExpect) Protocol(protocol string) *RequestExpect {
	e.req.Protocol(protocol)
	return e
}

func (e *RequestExpect) PortName(name string) *RequestExpect {
	if e.f != nil {
		e.req.Port(e.f.PortByName(name))
	}
	return e
}

func (e *RequestExpect) Port(port int) *RequestExpect {
	if e.f != nil {
		e.req.Port(port)
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

func (e *RequestExpect) Explain(explain ...any) *RequestExpect {
	e.explain = explain
	return e
}

type EnsureFunc func(*request.Response) bool

func (e *RequestExpect) Ensure(fns ...EnsureFunc) {
	ret, err := e.req.Do()
	if e.expectError {
		ExpectErrorWithOffset(1, err, e.explain...)
		return
	}
	ExpectNoErrorWithOffset(1, err, e.explain...)

	if len(fns) == 0 {
		if !bytes.Equal(e.expectResp, ret.Content) {
			flog.Tracef("Response info: %+v", ret)
		}
		ExpectEqualValuesWithOffset(1, string(ret.Content), string(e.expectResp), e.explain...)
	} else {
		for _, fn := range fns {
			ok := fn(ret)
			if !ok {
				flog.Tracef("Response info: %+v", ret)
			}
			ExpectTrueWithOffset(1, ok, e.explain...)
		}
	}
}

func (e *RequestExpect) Do() (*request.Response, error) {
	return e.req.Do()
}
