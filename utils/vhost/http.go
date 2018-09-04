// Copyright 2016 fatedier, fatedier@gmail.com
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package vhost

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	frpNet "github.com/fatedier/frp/utils/net"
	"github.com/fatedier/frp/utils/pool"
)

type HttpMuxer struct {
	*VhostMuxer
}

func GetHttpRequestInfo(c frpNet.Conn) (_ frpNet.Conn, _ map[string]string, err error) {
	reqInfoMap := make(map[string]string, 0)
	sc, rd := frpNet.NewShareConn(c)

	request, err := http.ReadRequest(bufio.NewReader(rd))
	if err != nil {
		return sc, reqInfoMap, err
	}
	// hostName
	tmpArr := strings.Split(request.Host, ":")
	reqInfoMap["Host"] = tmpArr[0]
	reqInfoMap["Path"] = request.URL.Path
	reqInfoMap["Scheme"] = request.URL.Scheme

	// Authorization
	authStr := request.Header.Get("Authorization")
	if authStr != "" {
		reqInfoMap["Authorization"] = authStr
	}
	request.Body.Close()
	return sc, reqInfoMap, nil
}

func NewHttpMuxer(listener frpNet.Listener, timeout time.Duration) (*HttpMuxer, error) {
	mux, err := NewVhostMuxer(listener, GetHttpRequestInfo, HttpAuthFunc, ModifyHttpRequest, timeout)
	return &HttpMuxer{mux}, err
}

func ModifyHttpRequest(c frpNet.Conn, rewriteHost string) (_ frpNet.Conn, err error) {
	sc, rd := frpNet.NewShareConn(c)
	var buff []byte
	remoteIP := strings.Split(c.RemoteAddr().String(), ":")[0]
	if buff, err = hostNameRewrite(rd, rewriteHost, remoteIP); err != nil {
		return sc, err
	}
	err = sc.WriteBuff(buff)
	return sc, err
}

func hostNameRewrite(request io.Reader, rewriteHost string, remoteIP string) (_ []byte, err error) {
	buf := pool.GetBuf(1024)
	defer pool.PutBuf(buf)

	var n int
	n, err = request.Read(buf)
	if err != nil {
		return
	}
	retBuffer, err := parseRequest(buf[:n], rewriteHost, remoteIP)
	return retBuffer, err
}

func parseRequest(org []byte, rewriteHost string, remoteIP string) (ret []byte, err error) {
	tp := bytes.NewBuffer(org)
	// First line: GET /index.html HTTP/1.0
	var b []byte
	if b, err = tp.ReadBytes('\n'); err != nil {
		return nil, err
	}
	req := new(http.Request)
	// we invoked ReadRequest in GetHttpHostname before, so we ignore error
	req.Method, req.RequestURI, req.Proto, _ = parseRequestLine(string(b))
	rawurl := req.RequestURI
	// CONNECT www.google.com:443 HTTP/1.1
	justAuthority := req.Method == "CONNECT" && !strings.HasPrefix(rawurl, "/")
	if justAuthority {
		rawurl = "http://" + rawurl
	}
	req.URL, _ = url.ParseRequestURI(rawurl)
	if justAuthority {
		// Strip the bogus "http://" back off.
		req.URL.Scheme = ""
	}

	//  RFC2616: first case
	//  GET /index.html HTTP/1.1
	//  Host: www.google.com
	if req.URL.Host == "" {
		var changedBuf []byte
		if rewriteHost != "" {
			changedBuf, err = changeHostName(tp, rewriteHost)
		}
		buf := new(bytes.Buffer)
		buf.Write(b)
		buf.WriteString(fmt.Sprintf("X-Forwarded-For: %s\r\n", remoteIP))
		buf.WriteString(fmt.Sprintf("X-Real-IP: %s\r\n", remoteIP))
		if len(changedBuf) == 0 {
			tp.WriteTo(buf)
		} else {
			buf.Write(changedBuf)
		}
		return buf.Bytes(), err
	}

	// RFC2616: second case
	// GET http://www.google.com/index.html HTTP/1.1
	// Host: doesntmatter
	// In this case, any Host line is ignored.
	if rewriteHost != "" {
		hostPort := strings.Split(req.URL.Host, ":")
		if len(hostPort) == 1 {
			req.URL.Host = rewriteHost
		} else if len(hostPort) == 2 {
			req.URL.Host = fmt.Sprintf("%s:%s", rewriteHost, hostPort[1])
		}
	}
	firstLine := req.Method + " " + req.URL.String() + " " + req.Proto
	buf := new(bytes.Buffer)
	buf.WriteString(firstLine)
	buf.WriteString(fmt.Sprintf("X-Forwarded-For: %s\r\n", remoteIP))
	buf.WriteString(fmt.Sprintf("X-Real-IP: %s\r\n", remoteIP))
	tp.WriteTo(buf)
	return buf.Bytes(), err
}

// parseRequestLine parses "GET /foo HTTP/1.1" into its three parts.
func parseRequestLine(line string) (method, requestURI, proto string, ok bool) {
	s1 := strings.Index(line, " ")
	s2 := strings.Index(line[s1+1:], " ")
	if s1 < 0 || s2 < 0 {
		return
	}
	s2 += s1 + 1
	return line[:s1], line[s1+1 : s2], line[s2+1:], true
}

func changeHostName(buff *bytes.Buffer, rewriteHost string) (_ []byte, err error) {
	retBuf := new(bytes.Buffer)

	peek := buff.Bytes()
	for len(peek) > 0 {
		i := bytes.IndexByte(peek, '\n')
		if i < 3 {
			// Not present (-1) or found within the next few bytes,
			// implying we're at the end ("\r\n\r\n" or "\n\n")
			return nil, err
		}
		kv := peek[:i]
		j := bytes.IndexByte(kv, ':')
		if j < 0 {
			return nil, fmt.Errorf("malformed MIME header line: " + string(kv))
		}
		if strings.Contains(strings.ToLower(string(kv[:j])), "host") {
			var hostHeader string
			portPos := bytes.IndexByte(kv[j+1:], ':')
			if portPos == -1 {
				hostHeader = fmt.Sprintf("Host: %s\r\n", rewriteHost)
			} else {
				hostHeader = fmt.Sprintf("Host: %s:%s\r\n", rewriteHost, kv[j+portPos+2:])
			}
			retBuf.WriteString(hostHeader)
			peek = peek[i+1:]
			break
		} else {
			retBuf.Write(peek[:i])
			retBuf.WriteByte('\n')
		}

		peek = peek[i+1:]
	}
	retBuf.Write(peek)
	return retBuf.Bytes(), err
}

func HttpAuthFunc(c frpNet.Conn, userName, passWord, authorization string) (bAccess bool, err error) {
	s := strings.SplitN(authorization, " ", 2)
	if len(s) != 2 {
		res := noAuthResponse()
		res.Write(c)
		return
	}
	b, err := base64.StdEncoding.DecodeString(s[1])
	if err != nil {
		return
	}
	pair := strings.SplitN(string(b), ":", 2)
	if len(pair) != 2 {
		return
	}
	if pair[0] != userName || pair[1] != passWord {
		return
	}
	return true, nil
}

func noAuthResponse() *http.Response {
	header := make(map[string][]string)
	header["WWW-Authenticate"] = []string{`Basic realm="Restricted"`}
	res := &http.Response{
		Status:     "401 Not authorized",
		StatusCode: 401,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     header,
	}
	return res
}
