package interceptor

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/fatedier/frp/pkg/util/cache"
	"github.com/fatedier/frp/pkg/util/log"
	"github.com/google/uuid"
)

var (
	defaultCacheDataInterceptor = NewCacheDataInterceptor(cache.DefaultCache)
)

type CacheDataInterceptor struct {
	c cache.Cacher
}

func NewCacheDataInterceptor(c cache.Cacher) *CacheDataInterceptor {
	return &CacheDataInterceptor{
		c: c,
	}
}

func (cc *CacheDataInterceptor) Intercept(req *http.Request, resp *http.Response, elapseMs int) {
	var (
		bodyReq  []byte
		bodyResp []byte
	)

	uniqueId := uuid.NewString()

	if req.Body != nil {
		bodyReq, _ = ioutil.ReadAll(req.Body)
		req.Body = ioutil.NopCloser(bytes.NewReader(bodyReq))
	}

	log.Debug("get request url: %v, path: %v, host: %v, header: %v, body: %v", req.URL.String(), req.URL.Path, req.Host, req.Header, string(bodyReq))
	rawReq := RawRequest{
		Method:        req.Method,
		URL:           req.URL.String(),
		Host:          req.Host,
		Header:        req.Header,
		Body:          bodyReq,
		ContentLength: req.ContentLength,
	}

	bodyResp, _ = ioutil.ReadAll(resp.Body)
	resp.Body = ioutil.NopCloser(bytes.NewReader(bodyResp))

	log.Debug("get response header: %v, body: %v, use: %v ms", resp.Header, string(bodyResp), elapseMs)
	rawResp := RawResponse{
		StatusCode:    resp.StatusCode,
		Header:        resp.Header,
		Body:          bodyResp,
		ContentLength: resp.ContentLength,
	}

	pair := Pair{
		Key: uniqueId,

		Req:  rawReq,
		Resp: rawResp,

		TrafficCaptureTime: time.Now().Unix(),

		ElapseMs: elapseMs,
	}

	pairByte, _ := json.Marshal(pair)
	if len(pairByte) <= 1024*1024 {
		// limit data size less than 1MB
		cc.c.Add(uniqueId, pair)
	}
}

type RawRequest struct {
	Method        string      `json:"method,omitempty"`
	URL           string      `json:"url,omitempty"`
	Host          string      `json:"host,omitempty"`
	Header        http.Header `json:"header,omitempty"`
	Body          []byte      `json:"body,omitempty"`
	ContentLength int64       `json:"content_length,omitempty"`
}

type RawResponse struct {
	StatusCode    int         `json:"status_code,omitempty"`
	Header        http.Header `json:"header,omitempty"`
	Body          []byte      `json:"body,omitempty"`
	ContentLength int64       `json:"content_length,omitempty"`
}

type Pair struct {
	Key string `json:"key,omitempty"`

	Req  RawRequest  `json:"req,omitempty"`
	Resp RawResponse `json:"resp,omitempty"`

	TrafficCaptureTime int64 `json:"traffic_capture_time,omitempty"`

	ElapseMs int `json:"elapse_ms,omitempty"`
}
