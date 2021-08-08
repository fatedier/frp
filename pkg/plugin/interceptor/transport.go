package interceptor

import (
	"net/http"
	"time"
)

type Interceptor func(req *http.Request, resp *http.Response, elapsMs int)

type TransportWrapper struct {
	tr           http.RoundTripper
	interceptors []Interceptor
}

func NewTransportWrapper(tr http.RoundTripper, is ...Interceptor) *TransportWrapper {
	tw := &TransportWrapper{
		tr: tr,
	}
	tw.interceptors = append(tw.interceptors, is...)

	return tw
}

func (tw *TransportWrapper) WithInterceptors(i ...Interceptor) {
	tw.interceptors = append(tw.interceptors, i...)
}

func (tw *TransportWrapper) WithCacheInterceptor() {
	tw.WithInterceptors(defaultCacheDataInterceptor.Intercept)
}

func (tw *TransportWrapper) RoundTrip(req *http.Request) (*http.Response, error) {
	startTime := time.Now()

	resp, err := tw.tr.RoundTrip(req)
	if err != nil {
		return resp, err
	}

	elapseMs := time.Since(startTime).Milliseconds()

	go func() {
		for _, op := range tw.interceptors {
			op(req, resp, int(elapseMs))
		}
	}()

	return resp, err
}
