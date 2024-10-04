package api

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/fatedier/frp/pkg/msg"
)

// Service sakurafrp api servie
type Service struct {
	Host url.URL
}

// NewService crate sakurafrp api servie
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
	values.Set("http_user", pMsg.HTTPUser)
	values.Set("http_pwd", pMsg.HTTPPwd)
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

// GetProxyLimit 获取隧道限速信息
func (s Service) GetProxyLimit(user string, timestamp int64, stk string) (inLimit, outLimit uint64, err error) {
	// 这部分就照之前的搬过去了，能跑就行x
	values := url.Values{}
	values.Set("action", "getlimit")
	values.Set("user", user)
	values.Set("timestamp", fmt.Sprintf("%d", timestamp))
	values.Set("apitoken", stk)
	s.Host.RawQuery = values.Encode()
	defer func(u *url.URL) {
		u.RawQuery = ""
	}(&s.Host)
	resp, err := http.Get(s.Host.String())
	if err != nil {
		return 0, 0, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, 0, err
	}

	er := &ErrHTTPStatus{}
	if err = json.Unmarshal(body, er); err != nil {
		return 0, 0, err
	}
	if er.Status != 200 {
		return 0, 0, er
	}

	response := &ResponseGetLimit{}
	if err = json.Unmarshal(body, response); err != nil {
		return 0, 0, err
	}

	// 这里直接返回 uint64 应该问题不大
	return response.MaxIn, response.MaxOut, nil

}

func BoolToString(val bool) (str string) {
	if val {
		return "true"
	}
	return "false"

}

type ErrHTTPStatus struct {
	Status int    `json:"status"`
	Text   string `json:"message"`
}

func (e ErrHTTPStatus) Error() string {
	t := time.Now()
	layout := "2006-01-02 15:04:05"
	str := t.Format(layout)
	switch e.Status {
	case 403:
		return fmt.Sprintf("HayFrpAPI return error.\n"+str+" [E] [api/hayfrp.go:465] [HayFrp] API返回错误，状态码：%d 文本：%s\n"+str+" [E] [hayfrp.go:470] [HayFrp] 根据状态码判定，问题为隧道被禁止或者无权启用隧道\n"+str+" [W] [hayfrp.go:475] [HayFrp] 客户端可能会在20s后重新尝试链接......\n"+str+" [W] [hayfrp.go:480] [HayFrp] 如果无法链接，请到面板检查隧道状态\n"+str+" [W] [hayfrp.go:485] [HayFrp] 若仍然出现问题，请截图至QQ群或和谐论坛反馈\n"+str+" [W] [hayfrp.go:490] [HayFrp] 注意，截图时请为Token打码，否则有泄露风险!", e.Status, e.Text)
	case 404:
		return fmt.Sprintf("HayFrpAPI return error.\n"+str+" [E] [api/hayfrp.go:466] [HayFrp] API返回错误，状态码：%d 文本：%s\n"+str+" [E] [hayfrp.go:471] [HayFrp] 根据状态码判定，问题为没有找到隧道或者API出现故障\n"+str+" [W] [hayfrp.go:476] [HayFrp] 客户端可能会在20s后重新尝试链接......\n"+str+" [W] [hayfrp.go:481] [HayFrp] 如果无法链接，请到面板检查隧道状态\n"+str+" [W] [hayfrp.go:486] [HayFrp] 若仍然出现问题，请截图至QQ群或和谐论坛反馈\n"+str+" [W] [hayfrp.go:491] [HayFrp] 注意，截图时请为Token打码，否则有泄露风险!", e.Status, e.Text)
	case 520:
		return fmt.Sprintf("HayFrpAPI return error.\n"+str+" [E] [api/hayfrp.go:467] [HayFrp] API返回错误，状态码：%d 文本：%s\n"+str+" [E] [hayfrp.go:472] [HayFrp] 根据状态码判定，问题为API网关出现故障\n"+str+" [W] [hayfrp.go] [HayFrp:477] 客户端可能会在20s后重新尝试链接......\n"+str+" [W] [hayfrp.go:482] [HayFrp] 如果无法链接，请到面板检查隧道状态\n"+str+" [W] [hayfrp.go:487] [HayFrp] 若仍然出现问题，请截图至QQ群或和谐论坛反馈\n"+str+" [W] [hayfrp.go:492] [HayFrp] 注意，截图时请为Token打码，否则有泄露风险!", e.Status, e.Text)
	case 503:
		return fmt.Sprintf("HayFrpAPI return error.\n"+str+" [E] [api/hayfrp.go:468] [HayFrp] API返回错误，状态码：%d 文本：%s\n"+str+" [E] [hayfrp.go:473] [HayFrp] 根据状态码判定，问题为API请求了过大进行限流\n"+str+" [W] [hayfrp.go:478] [HayFrp] 客户端可能会在20s后重新尝试链接......\n"+str+" [W] [hayfrp.go:483] [HayFrp] 如果无法链接，请到面板检查隧道状态\n"+str+" [W] [hayfrp.go:488] [HayFrp] 若仍然出现问题，请截图至QQ群或和谐论坛反馈\n"+str+" [W] [hayfrp.go:493] [HayFrp] 注意，截图时请为Token打码，否则有泄露风险!", e.Status, e.Text)
	default:
		return fmt.Sprintf("HayFrpAPI return error.\n"+str+" [E] [api/hayfrp.go:469] [HayFrp] API返回错误，状态码：%d 文本：%s\n"+str+" [E] [hayfrp.go:474] [HayFrp] 目前HayFrp无法状态码判定问题......\n"+str+" [W] [hayfrp.go:479] [HayFrp] 客户端可能会在20s后重新尝试链接......\n"+str+" [W] [hayfrp.go:484] [HayFrp] 如果无法链接，请到面板检查隧道状态\n"+str+" [W] [hayfrp.go:489] [HayFrp] 若仍然出现问题，请截图至QQ群或和谐论坛反馈\n"+str+" [W] [hayfrp.go:494] [HayFrp] 注意，截图时请为Token打码，否则有泄露风险!", e.Status, e.Text)
	}
}

type ResponseGetLimit struct {
	MaxIn  uint64 `json:"max-in"`
	MaxOut uint64 `json:"max-out"`
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
