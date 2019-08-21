package api

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	
	"github.com/fatedier/frp/models/msg"
)

type Service struct {
	Host url.URL
}

func NewService(host string) (s *Service, err error) {
	u, err := url.Parse(host)
	if err != nil {
		return
	}
	return &Service{*u}, nil
}

// CheckToken 校验客户端 token
func (s Service) CheckToken(user string, token string, timestamp int64, stk string) (ok bool, err error) {
	values := url.Values{}
	values.Set("action", "checktoken")
	values.Set("user", user)
	values.Set("token", token)
	values.Set("timestamp", fmt.Sprintf("%d", timestamp))
	values.Set("apitoken", stk)
	s.Host.RawQuery = values.Encode()
	defer func(u *url.URL) {
		u.RawQuery = ""
	}(&s.Host)
	resp, err := http.Get(s.Host.String())
	if err != nil {
		return false, err
	}
	if resp.StatusCode != http.StatusOK {
		return false, ErrHTTPStatus{
			Status: resp.StatusCode,
			Text:   resp.Status,
		}
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}
	response := ResponseCheckToken{}
	if err = json.Unmarshal(body, &response); err != nil {
		return false, err
	}
	if !response.Success {
		return false, ErrCheckTokenFail{response.Message}
	}
	return true, nil
}

// CheckProxy 校验客户端代理
func (s Service) CheckProxy(user string, pMsg *msg.NewProxy, timestamp int64, stk string) (ok bool, err error) {

	domains, err := json.Marshal(pMsg.CustomDomains)
	if err != nil {
		return false, err
	}
	
	headers, err := json.Marshal(pMsg.Headers)
	if err != nil {
		return false, err
	}
	
	locations, err := json.Marshal(pMsg.Locations)
	if err != nil {
		return false, err
	}
	
	values := url.Values{}
	
	// API Basic
	values.Set("action", "checkproxy")
	values.Set("user", user)
	values.Set("timestamp", fmt.Sprintf("%d", timestamp))
	values.Set("apitoken", stk)
	
	// Proxies basic info
	values.Set("proxy_name", pMsg.ProxyName)
	values.Set("proxy_type", pMsg.ProxyType)
	values.Set("use_encryption", BoolToString(pMsg.UseEncryption))
	values.Set("use_compression", BoolToString(pMsg.UseCompression))
	
	// Http Proxies
	values.Set("domain", string(domains))
	values.Set("subdomain", pMsg.SubDomain)
	
	// Headers
	values.Set("locations", string(locations))
	values.Set("http_user", pMsg.HttpUser)
	values.Set("http_pwd", pMsg.HttpPwd)
	values.Set("host_header_rewrite", pMsg.HostHeaderRewrite)
	values.Set("headers", string(headers))
	
	// Tcp & Udp & Stcp
	values.Set("remote_port", strconv.Itoa(pMsg.RemotePort))
	
	// Stcp & Xtcp
	values.Set("sk", pMsg.Sk)
	
	// Load balance
	values.Set("group", pMsg.Group)
	values.Set("group_key", pMsg.GroupKey)
	
	s.Host.RawQuery = values.Encode()
	defer func(u *url.URL) {
		u.RawQuery = ""
	}(&s.Host)
	resp, err := http.Get(s.Host.String())
	if err != nil {
		return false, err
	}
	if resp.StatusCode != http.StatusOK {
		return false, ErrHTTPStatus{
			Status: resp.StatusCode,
			Text:   resp.Status,
		}
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}
	response := ResponseCheckProxy{}
	if err = json.Unmarshal(body, &response); err != nil {
		return false, err
	}
	if !response.Success {
		return false, ErrCheckProxyFail{response.Message}
	}
	return true, nil
}

func BoolToString(val bool) (str string) {
	if val {
		return "true"
	} else {
		return "false"
	}
}

type ErrHTTPStatus struct {
	Status int    `json:"status"`
	Text   string `json:"massage"`
}

func (e ErrHTTPStatus) Error() string {
	return fmt.Sprintf("%s", e.Text)
}

type ResponseCheckToken struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type ResponseCheckProxy struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type ErrCheckTokenFail struct {
	Message string
}

type ErrCheckProxyFail struct {
	Message string
}

func (e ErrCheckTokenFail) Error() string {
	return e.Message
}

func (e ErrCheckProxyFail) Error() string {
	return e.Message
}
