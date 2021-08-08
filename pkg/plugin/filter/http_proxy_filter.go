package filter

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"time"

	"github.com/fatedier/frp/pkg/config"
	"github.com/fatedier/frp/pkg/plugin/interceptor"
	"github.com/fatedier/frp/pkg/util/listener"
	frpNet "github.com/fatedier/frp/pkg/util/net"
)

const (
	StreamFilterName = "StreamFilterName"
)

type HTTPStreamFilter struct {
	l *listener.Listener
	s *http.Server
}

func NewHTTPStreamFilter(filterConf config.LocalSvrConf, params map[string]string) (*HTTPStreamFilter, error) {
	listener := listener.NewProxyListener()

	p := &HTTPStreamFilter{
		l: listener,
	}

	localAddr := fmt.Sprintf("%v:%v", filterConf.LocalIP, filterConf.LocalPort)

	tr := interceptor.NewTransportWrapper(&http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return net.Dial("tcp", localAddr)
		},
		ResponseHeaderTimeout: time.Duration(90) * time.Second,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
	})
	tr.WithCacheInterceptor()

	rp := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = "http"
			req.URL.Host = localAddr
			req.Host = localAddr
		},
		Transport: tr,
	}

	p.s = &http.Server{
		Handler: rp,
	}

	go p.s.Serve(listener)

	return p, nil
}

func (p *HTTPStreamFilter) Handle(conn io.ReadWriteCloser, realConn net.Conn) {
	wrapConn := frpNet.WrapReadWriteCloserToConn(conn, realConn)
	p.l.PutConn(wrapConn)
}

func (p *HTTPStreamFilter) Name() string {
	return StreamFilterName
}

func (p *HTTPStreamFilter) Close() error {
	if err := p.s.Close(); err != nil {
		return err
	}
	return nil
}
