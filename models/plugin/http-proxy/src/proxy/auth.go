package proxy

import (
	"encoding/base64"
	"errors"
	"net/http"
	"strings"

	"github.com/fatedier/frp/utils/log"
)

/*
http://www.checkupdown.com/status/E407_zh.html
HTTP 407 错误 - 要求代理身份验证 (Proxy authentication required)
您的 Web 服务器认为客户端（如您的浏览器或我们的 CheckUpDown 机器人）发送的 HTTP 数据流是正确的，但访问该网址资源需要事先经过一个代理服务器，而该代理服务器所需的身份验证尚未提供。 这通常意味着您必须首先登录（输入用户名和密码）代理服务器。

通过浏览器检测到的 407 错误往往可以通过选择略有不同的导航途径访问该网址来解决， 如先访问该代理服务器的其他网址 。 您的互联网服务供应商 (ISP) 应该能够解释该代理服务器在其安全设置中的作用，以及如何使用。

*/

// https://developer.mozilla.org/en-US/docs/Web/HTTP/Status/407
/*
HTTP/1.1 407 Proxy Authentication Required
Date: Wed, 21 Oct 2015 07:28:00 GMT
Proxy-Authenticate: Basic realm="Access to internal site"
*/
var HTTP_407 = []byte("HTTP/1.1 407 Proxy Authorization Required\r\nProxy-Authenticate: Basic realm=\"Access to internal site\"\r\n\r\n")

// 鉴权方法
func (proxy *ProxyServer) Auth(rw http.ResponseWriter, req *http.Request) bool {
	var err error
	if cnfg.Auth == true { // 代理服务器登入认证
		if proxy.Name, err = proxy.auth(rw, req); err != nil {
			log.Debug("%s can not successfully access %v", proxy.Name, err)
			return true
		}
	} else {
		proxy.Name = "default-proxy"
	}

	// log.Info("%s successfully log in!", proxy.Name)
	return false
}

func (proxy *ProxyServer) auth(rw http.ResponseWriter, req *http.Request) (string, error) {

	// get header
	// Proxy-Authorization: Basic YWxhZGRpbjpvcGVuc2VzYW1l
	auth := req.Header.Get("Proxy-Authorization")
	auth = strings.Replace(auth, "Basic ", "", 1)

	if auth == "" {
		writeResp(rw, HTTP_407)
		return "", errors.New("Need Proxy Authorization!")
	}

	// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Proxy-Authorization
	data, err := base64.StdEncoding.DecodeString(auth)
	if err != nil {
		log.Debug("when decoding %v, got an error of %v", auth, err)
		return "", errors.New("Fail to decoding Proxy-Authorization")
	}

	var user, passwd string

	// username:password
	UserPasswdPair := strings.Split(string(data), ":")
	if len(UserPasswdPair) != 2 {
		writeResp(rw, HTTP_407)
		return "", errors.New("Fail to log in")
	} else {
		user = UserPasswdPair[0]
		passwd = UserPasswdPair[1]
	}

	if check(user, passwd) == false {
		writeResp(rw, HTTP_407)
		return "", errors.New("Fail to log in")
	}
	return user, nil
}

func writeResp(rw http.ResponseWriter, data []byte) error {
	_, err := rw.Write(data)
	if err != nil {
		log.Error("fail to write data to response")
		return errors.New("InternalServerError")
	}
	return nil
}

// 验证浏览器输入的proxy是否合法
func check(User, passwd string) bool {
	if User != "" && passwd != "" && cnfg.User[User] == passwd {
		return true
	}
	return false
}
