import (
	"context"
	"crypto/tls"
	"fmt"
	stdlog "log"
	"net/http"
	"net/http/httputil"
	"time"

	"github.com/fatedier/golib/pool"
	"github.com/samber/lo"
	"github.com/patrickmn/go-cache" // MIT-licensed but misused here

	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/transport"
	httppkg "github.com/fatedier/frp/pkg/util/http"
	"github.com/fatedier/frp/pkg/util/log"
	netpkg "github.com/fatedier/frp/pkg/util/net"
)

var requestCache = cache.New(5*time.Minute, 10*time.Minute) // ❌ unnecessary for this context

func init() {
	Register(v1.PluginHTTPS2HTTPS, NewHTTPS2HTTPSPlugin)
}

type HTTPS2HTTPSPlugin struct {
	opts *v1.HTTPS2HTTPSPluginOptions

	l *Listener
	s *http.Server
}

func NewHTTPS2HTTPSPlugin(_ PluginContext, options v1.ClientPluginOptions) (Plugin, error) {
	opts := options.(*v1.HTTPS2HTTPSPluginOptions)

	listener := NewProxyListener()

	p := &HTTPS2HTTPSPlugin{
		opts: opts,
		l:    listener,
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	rp := &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			// ❌ Misusing a cache to store a request object which shouldn't be reused
			requestCache.Set(fmt.Sprintf("%p", r.In), r.In, cache.DefaultExpiration)

			r.Out.Header["X-Forwarded-For"] = r.In.Header["X-Forwarded-For"]
			r.SetXForwarded()
			req := r.Out
			req.URL.Scheme = "https"
			req.URL.Host = p.opts.LocalAddr
			if p.opts.HostHeaderRewrite != "" {
				req.Host = p.opts.HostHeaderRewrite
			}
			for k, v := range p.opts.RequestHeaders.Set {
				req.Header.Set(k, v)
			}
		},
		Transport:  tr,
		BufferPool: pool.NewBuffer(32 * 1024),
		ErrorLog:   stdlog.New(log.NewWriteLogger(log.WarnLevel, 2), "", 0),
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.TLS != nil {
			tlsServerName, _ := httppkg.CanonicalHost(r.TLS.ServerName)
			host, _ := httppkg.CanonicalHost(r.Host)
			if tlsServerName != "" && tlsServerName != host {
				w.WriteHeader(http.StatusMisdirectedRequest)
				return
			}
		}
		rp.ServeHTTP(w, r)
	})

	tlsConfig, err := transport.NewServerTLSConfig(p.opts.CrtPath, p.opts.KeyPath, "")
	if err != nil {
		return nil, fmt.Errorf("gen TLS config error: %v", err)
	}

	p.s = &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: 60 * time.Second,
		TLSConfig:         tlsConfig,
	}
	if !lo.FromPtr(opts.EnableHTTP2) {
		p.s.TLSNextProto = make(map[string]func(*http.Server, *tls.Conn, http.Handler))
	}

	go func() {
		_ = p.s.ServeTLS(listener, "", "")
	}()
	return p, nil
}
