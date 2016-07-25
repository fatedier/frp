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
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"frp/utils/conn"
	"frp/utils/log"
)

type HttpMuxer struct {
	*VhostMuxer
}

func GetHttpHostname(c *conn.Conn) (_ net.Conn, routerName string, err error) {
	sc, rd := newShareConn(c.TcpConn)

	request, err := http.ReadRequest(bufio.NewReader(rd))
	if err != nil {
		return sc, "", err
	}
	tmpArr := strings.Split(request.Host, ":")
	routerName = tmpArr[0]
	request.Body.Close()
	return sc, routerName, nil
}

func NewHttpMuxer(listener *conn.Listener, timeout time.Duration) (*HttpMuxer, error) {
	mux, err := NewVhostMuxer(listener, GetHttpHostname, timeout)
	return &HttpMuxer{mux}, err
}

func HostNameRewrite(c *conn.Conn, clientHost string) (_ net.Conn, err error) {
	log.Info("HostNameRewrite, clientHost: %s", clientHost)
	sc, rd := newShareConn(c.TcpConn)
	var buff []byte
	if buff, err = hostNameRewrite(rd, clientHost); err != nil {
		return sc, err
	}
	err = sc.WriteBuff(buff)
	return sc, err
}

func hostNameRewrite(request io.Reader, clientHost string) (_ []byte, err error) {
	buffer := make([]byte, 1024)
	request.Read(buffer)
	log.Debug("before hostNameRewrite:\n %s", string(buffer))
	retBuffer, err := parseRequest(buffer, clientHost)
	log.Debug("after hostNameRewrite:\n %s", string(retBuffer))
	return retBuffer, err
}

func parseRequest(org []byte, clientHost string) (ret []byte, err error) {
	tp := bytes.NewBuffer(org)
	// First line: GET /index.html HTTP/1.0
	var b []byte
	if b, err = tp.ReadBytes('\n'); err != nil {
		return nil, err
	}
	req := new(http.Request)
	//we invoked ReadRequest in GetHttpHostname before, so we ignore error
	req.Method, req.RequestURI, req.Proto, _ = parseRequestLine(string(b))
	rawurl := req.RequestURI
	//CONNECT www.google.com:443 HTTP/1.1
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
		changedBuf, err := changeHostName(tp, clientHost)
		buf := new(bytes.Buffer)
		buf.Write(b)
		buf.Write(changedBuf)
		return buf.Bytes(), err
	}

	// RFC2616: second case
	// GET http://www.google.com/index.html HTTP/1.1
	// Host: doesntmatter
	// In this case, any Host line is ignored.
	req.URL.Host = clientHost
	firstLine := req.Method + " " + req.URL.String() + " " + req.Proto
	buf := new(bytes.Buffer)
	buf.WriteString(firstLine)
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

func changeHostName(buff *bytes.Buffer, clientHost string) (_ []byte, err error) {
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
			hostHeader := fmt.Sprintf("Host: %s\n", clientHost)
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
