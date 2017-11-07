// Copyright 2014 beego Author. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package httplib is used as http.Client
// Usage:
//
// import "github.com/astaxie/beego/httplib"
//
//	b := httplib.Post("http://beego.me/")
//	b.Param("username","astaxie")
//	b.Param("password","123456")
//	b.PostFile("uploadfile1", "httplib.pdf")
//	b.PostFile("uploadfile2", "httplib.txt")
//	str, err := b.String()
//	if err != nil {
//		t.Fatal(err)
//	}
//	fmt.Println(str)
//
//  more docs http://beego.me/docs/module/httplib.md
package httplib

import (
	"bytes"
	"compress/gzip"
	"crypto/tls"
	"encoding/json"
	"encoding/xml"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

var defaultSetting = BeegoHTTPSettings{
	UserAgent:        "beegoServer",
	ConnectTimeout:   60 * time.Second,
	ReadWriteTimeout: 60 * time.Second,
	Gzip:             true,
	DumpBody:         true,
}

var defaultCookieJar http.CookieJar
var settingMutex sync.Mutex

// createDefaultCookie creates a global cookiejar to store cookies.
func createDefaultCookie() {
	settingMutex.Lock()
	defer settingMutex.Unlock()
	defaultCookieJar, _ = cookiejar.New(nil)
}

// SetDefaultSetting Overwrite default settings
func SetDefaultSetting(setting BeegoHTTPSettings) {
	settingMutex.Lock()
	defer settingMutex.Unlock()
	defaultSetting = setting
}

// NewBeegoRequest return *BeegoHttpRequest with specific method
func NewBeegoRequest(rawurl, method string) *BeegoHTTPRequest {
	var resp http.Response
	u, err := url.Parse(rawurl)
	if err != nil {
		log.Println("Httplib:", err)
	}
	req := http.Request{
		URL:        u,
		Method:     method,
		Header:     make(http.Header),
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
	}
	return &BeegoHTTPRequest{
		url:     rawurl,
		req:     &req,
		params:  map[string][]string{},
		files:   map[string]string{},
		setting: defaultSetting,
		resp:    &resp,
	}
}

// Get returns *BeegoHttpRequest with GET method.
func Get(url string) *BeegoHTTPRequest {
	return NewBeegoRequest(url, "GET")
}

// Post returns *BeegoHttpRequest with POST method.
func Post(url string) *BeegoHTTPRequest {
	return NewBeegoRequest(url, "POST")
}

// Put returns *BeegoHttpRequest with PUT method.
func Put(url string) *BeegoHTTPRequest {
	return NewBeegoRequest(url, "PUT")
}

// Delete returns *BeegoHttpRequest DELETE method.
func Delete(url string) *BeegoHTTPRequest {
	return NewBeegoRequest(url, "DELETE")
}

// Head returns *BeegoHttpRequest with HEAD method.
func Head(url string) *BeegoHTTPRequest {
	return NewBeegoRequest(url, "HEAD")
}

// BeegoHTTPSettings is the http.Client setting
type BeegoHTTPSettings struct {
	ShowDebug        bool
	UserAgent        string
	ConnectTimeout   time.Duration
	ReadWriteTimeout time.Duration
	TLSClientConfig  *tls.Config
	Proxy            func(*http.Request) (*url.URL, error)
	Transport        http.RoundTripper
	CheckRedirect    func(req *http.Request, via []*http.Request) error
	EnableCookie     bool
	Gzip             bool
	DumpBody         bool
	Retries          int // if set to -1 means will retry forever
}

// BeegoHTTPRequest provides more useful methods for requesting one url than http.Request.
type BeegoHTTPRequest struct {
	url     string
	req     *http.Request
	params  map[string][]string
	files   map[string]string
	setting BeegoHTTPSettings
	resp    *http.Response
	body    []byte
	dump    []byte
}

// GetRequest return the request object
func (b *BeegoHTTPRequest) GetRequest() *http.Request {
	return b.req
}

// Setting Change request settings
func (b *BeegoHTTPRequest) Setting(setting BeegoHTTPSettings) *BeegoHTTPRequest {
	b.setting = setting
	return b
}

// SetBasicAuth sets the request's Authorization header to use HTTP Basic Authentication with the provided username and password.
func (b *BeegoHTTPRequest) SetBasicAuth(username, password string) *BeegoHTTPRequest {
	b.req.SetBasicAuth(username, password)
	return b
}

// SetEnableCookie sets enable/disable cookiejar
func (b *BeegoHTTPRequest) SetEnableCookie(enable bool) *BeegoHTTPRequest {
	b.setting.EnableCookie = enable
	return b
}

// SetUserAgent sets User-Agent header field
func (b *BeegoHTTPRequest) SetUserAgent(useragent string) *BeegoHTTPRequest {
	b.setting.UserAgent = useragent
	return b
}

// Debug sets show debug or not when executing request.
func (b *BeegoHTTPRequest) Debug(isdebug bool) *BeegoHTTPRequest {
	b.setting.ShowDebug = isdebug
	return b
}

// Retries sets Retries times.
// default is 0 means no retried.
// -1 means retried forever.
// others means retried times.
func (b *BeegoHTTPRequest) Retries(times int) *BeegoHTTPRequest {
	b.setting.Retries = times
	return b
}

// DumpBody setting whether need to Dump the Body.
func (b *BeegoHTTPRequest) DumpBody(isdump bool) *BeegoHTTPRequest {
	b.setting.DumpBody = isdump
	return b
}

// DumpRequest return the DumpRequest
func (b *BeegoHTTPRequest) DumpRequest() []byte {
	return b.dump
}

// SetTimeout sets connect time out and read-write time out for BeegoRequest.
func (b *BeegoHTTPRequest) SetTimeout(connectTimeout, readWriteTimeout time.Duration) *BeegoHTTPRequest {
	b.setting.ConnectTimeout = connectTimeout
	b.setting.ReadWriteTimeout = readWriteTimeout
	return b
}

// SetTLSClientConfig sets tls connection configurations if visiting https url.
func (b *BeegoHTTPRequest) SetTLSClientConfig(config *tls.Config) *BeegoHTTPRequest {
	b.setting.TLSClientConfig = config
	return b
}

// Header add header item string in request.
func (b *BeegoHTTPRequest) Header(key, value string) *BeegoHTTPRequest {
	b.req.Header.Set(key, value)
	return b
}

// SetHost set the request host
func (b *BeegoHTTPRequest) SetHost(host string) *BeegoHTTPRequest {
	b.req.Host = host
	return b
}

// SetProtocolVersion Set the protocol version for incoming requests.
// Client requests always use HTTP/1.1.
func (b *BeegoHTTPRequest) SetProtocolVersion(vers string) *BeegoHTTPRequest {
	if len(vers) == 0 {
		vers = "HTTP/1.1"
	}

	major, minor, ok := http.ParseHTTPVersion(vers)
	if ok {
		b.req.Proto = vers
		b.req.ProtoMajor = major
		b.req.ProtoMinor = minor
	}

	return b
}

// SetCookie add cookie into request.
func (b *BeegoHTTPRequest) SetCookie(cookie *http.Cookie) *BeegoHTTPRequest {
	b.req.Header.Add("Cookie", cookie.String())
	return b
}

// SetTransport set the setting transport
func (b *BeegoHTTPRequest) SetTransport(transport http.RoundTripper) *BeegoHTTPRequest {
	b.setting.Transport = transport
	return b
}

// SetProxy set the http proxy
// example:
//
//	func(req *http.Request) (*url.URL, error) {
// 		u, _ := url.ParseRequestURI("http://127.0.0.1:8118")
// 		return u, nil
// 	}
func (b *BeegoHTTPRequest) SetProxy(proxy func(*http.Request) (*url.URL, error)) *BeegoHTTPRequest {
	b.setting.Proxy = proxy
	return b
}

// SetCheckRedirect specifies the policy for handling redirects.
//
// If CheckRedirect is nil, the Client uses its default policy,
// which is to stop after 10 consecutive requests.
func (b *BeegoHTTPRequest) SetCheckRedirect(redirect func(req *http.Request, via []*http.Request) error) *BeegoHTTPRequest {
	b.setting.CheckRedirect = redirect
	return b
}

// Param adds query param in to request.
// params build query string as ?key1=value1&key2=value2...
func (b *BeegoHTTPRequest) Param(key, value string) *BeegoHTTPRequest {
	if param, ok := b.params[key]; ok {
		b.params[key] = append(param, value)
	} else {
		b.params[key] = []string{value}
	}
	return b
}

// PostFile add a post file to the request
func (b *BeegoHTTPRequest) PostFile(formname, filename string) *BeegoHTTPRequest {
	b.files[formname] = filename
	return b
}

// Body adds request raw body.
// it supports string and []byte.
func (b *BeegoHTTPRequest) Body(data interface{}) *BeegoHTTPRequest {
	switch t := data.(type) {
	case string:
		bf := bytes.NewBufferString(t)
		b.req.Body = ioutil.NopCloser(bf)
		b.req.ContentLength = int64(len(t))
	case []byte:
		bf := bytes.NewBuffer(t)
		b.req.Body = ioutil.NopCloser(bf)
		b.req.ContentLength = int64(len(t))
	}
	return b
}

// JSONBody adds request raw body encoding by JSON.
func (b *BeegoHTTPRequest) JSONBody(obj interface{}) (*BeegoHTTPRequest, error) {
	if b.req.Body == nil && obj != nil {
		byts, err := json.Marshal(obj)
		if err != nil {
			return b, err
		}
		b.req.Body = ioutil.NopCloser(bytes.NewReader(byts))
		b.req.ContentLength = int64(len(byts))
		b.req.Header.Set("Content-Type", "application/json")
	}
	return b, nil
}

func (b *BeegoHTTPRequest) buildURL(paramBody string) {
	// build GET url with query string
	if b.req.Method == "GET" && len(paramBody) > 0 {
		if strings.Index(b.url, "?") != -1 {
			b.url += "&" + paramBody
		} else {
			b.url = b.url + "?" + paramBody
		}
		return
	}

	// build POST/PUT/PATCH url and body
	if (b.req.Method == "POST" || b.req.Method == "PUT" || b.req.Method == "PATCH") && b.req.Body == nil {
		// with files
		if len(b.files) > 0 {
			pr, pw := io.Pipe()
			bodyWriter := multipart.NewWriter(pw)
			go func() {
				for formname, filename := range b.files {
					fileWriter, err := bodyWriter.CreateFormFile(formname, filename)
					if err != nil {
						log.Println("Httplib:", err)
					}
					fh, err := os.Open(filename)
					if err != nil {
						log.Println("Httplib:", err)
					}
					//iocopy
					_, err = io.Copy(fileWriter, fh)
					fh.Close()
					if err != nil {
						log.Println("Httplib:", err)
					}
				}
				for k, v := range b.params {
					for _, vv := range v {
						bodyWriter.WriteField(k, vv)
					}
				}
				bodyWriter.Close()
				pw.Close()
			}()
			b.Header("Content-Type", bodyWriter.FormDataContentType())
			b.req.Body = ioutil.NopCloser(pr)
			return
		}

		// with params
		if len(paramBody) > 0 {
			b.Header("Content-Type", "application/x-www-form-urlencoded")
			b.Body(paramBody)
		}
	}
}

func (b *BeegoHTTPRequest) getResponse() (*http.Response, error) {
	if b.resp.StatusCode != 0 {
		return b.resp, nil
	}
	resp, err := b.DoRequest()
	if err != nil {
		return nil, err
	}
	b.resp = resp
	return resp, nil
}

// DoRequest will do the client.Do
func (b *BeegoHTTPRequest) DoRequest() (resp *http.Response, err error) {
	var paramBody string
	if len(b.params) > 0 {
		var buf bytes.Buffer
		for k, v := range b.params {
			for _, vv := range v {
				buf.WriteString(url.QueryEscape(k))
				buf.WriteByte('=')
				buf.WriteString(url.QueryEscape(vv))
				buf.WriteByte('&')
			}
		}
		paramBody = buf.String()
		paramBody = paramBody[0 : len(paramBody)-1]
	}

	b.buildURL(paramBody)
	url, err := url.Parse(b.url)
	if err != nil {
		return nil, err
	}

	b.req.URL = url

	trans := b.setting.Transport

	if trans == nil {
		// create default transport
		trans = &http.Transport{
			TLSClientConfig:     b.setting.TLSClientConfig,
			Proxy:               b.setting.Proxy,
			Dial:                TimeoutDialer(b.setting.ConnectTimeout, b.setting.ReadWriteTimeout),
			MaxIdleConnsPerHost: -1,
		}
	} else {
		// if b.transport is *http.Transport then set the settings.
		if t, ok := trans.(*http.Transport); ok {
			if t.TLSClientConfig == nil {
				t.TLSClientConfig = b.setting.TLSClientConfig
			}
			if t.Proxy == nil {
				t.Proxy = b.setting.Proxy
			}
			if t.Dial == nil {
				t.Dial = TimeoutDialer(b.setting.ConnectTimeout, b.setting.ReadWriteTimeout)
			}
		}
	}

	var jar http.CookieJar
	if b.setting.EnableCookie {
		if defaultCookieJar == nil {
			createDefaultCookie()
		}
		jar = defaultCookieJar
	}

	client := &http.Client{
		Transport: trans,
		Jar:       jar,
	}

	if b.setting.UserAgent != "" && b.req.Header.Get("User-Agent") == "" {
		b.req.Header.Set("User-Agent", b.setting.UserAgent)
	}

	if b.setting.CheckRedirect != nil {
		client.CheckRedirect = b.setting.CheckRedirect
	}

	if b.setting.ShowDebug {
		dump, err := httputil.DumpRequest(b.req, b.setting.DumpBody)
		if err != nil {
			log.Println(err.Error())
		}
		b.dump = dump
	}
	// retries default value is 0, it will run once.
	// retries equal to -1, it will run forever until success
	// retries is setted, it will retries fixed times.
	for i := 0; b.setting.Retries == -1 || i <= b.setting.Retries; i++ {
		resp, err = client.Do(b.req)
		if err == nil {
			break
		}
	}
	return resp, err
}

// String returns the body string in response.
// it calls Response inner.
func (b *BeegoHTTPRequest) String() (string, error) {
	data, err := b.Bytes()
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// Bytes returns the body []byte in response.
// it calls Response inner.
func (b *BeegoHTTPRequest) Bytes() ([]byte, error) {
	if b.body != nil {
		return b.body, nil
	}
	resp, err := b.getResponse()
	if err != nil {
		return nil, err
	}
	if resp.Body == nil {
		return nil, nil
	}
	defer resp.Body.Close()
	if b.setting.Gzip && resp.Header.Get("Content-Encoding") == "gzip" {
		reader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, err
		}
		b.body, err = ioutil.ReadAll(reader)
	} else {
		b.body, err = ioutil.ReadAll(resp.Body)
	}
	return b.body, err
}

// ToFile saves the body data in response to one file.
// it calls Response inner.
func (b *BeegoHTTPRequest) ToFile(filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	resp, err := b.getResponse()
	if err != nil {
		return err
	}
	if resp.Body == nil {
		return nil
	}
	defer resp.Body.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}

// ToJSON returns the map that marshals from the body bytes as json in response .
// it calls Response inner.
func (b *BeegoHTTPRequest) ToJSON(v interface{}) error {
	data, err := b.Bytes()
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

// ToXML returns the map that marshals from the body bytes as xml in response .
// it calls Response inner.
func (b *BeegoHTTPRequest) ToXML(v interface{}) error {
	data, err := b.Bytes()
	if err != nil {
		return err
	}
	return xml.Unmarshal(data, v)
}

// Response executes request client gets response mannually.
func (b *BeegoHTTPRequest) Response() (*http.Response, error) {
	return b.getResponse()
}

// TimeoutDialer returns functions of connection dialer with timeout settings for http.Transport Dial field.
func TimeoutDialer(cTimeout time.Duration, rwTimeout time.Duration) func(net, addr string) (c net.Conn, err error) {
	return func(netw, addr string) (net.Conn, error) {
		conn, err := net.DialTimeout(netw, addr, cTimeout)
		if err != nil {
			return nil, err
		}
		err = conn.SetDeadline(time.Now().Add(rwTimeout))
		return conn, err
	}
}
