package request

import (
	"fmt"
	"net"
	"time"

	libnet "github.com/fatedier/golib/net"
)

type Request struct {
	protocol  string
	addr      string
	port      int
	body      []byte
	timeout   time.Duration
	proxyURL  string
	proxyHost string
}

func New() *Request {
	return &Request{
		protocol: "tcp",
	}
}

func (r *Request) Protocol(protocol string) *Request {
	r.protocol = protocol
	return r
}

func (r *Request) TCP() *Request {
	r.protocol = "tcp"
	return r
}

func (r *Request) UDP() *Request {
	r.protocol = "udp"
	return r
}

func (r *Request) Proxy(url, host string) *Request {
	r.proxyURL = url
	r.proxyHost = host
	return r
}

func (r *Request) Addr(addr string) *Request {
	r.addr = addr
	return r
}

func (r *Request) Port(port int) *Request {
	r.port = port
	return r
}

func (r *Request) Timeout(timeout time.Duration) *Request {
	r.timeout = timeout
	return r
}

func (r *Request) Body(content []byte) *Request {
	r.body = content
	return r
}

func (r *Request) Do() ([]byte, error) {
	var (
		conn net.Conn
		err  error
	)
	if len(r.proxyURL) > 0 {
		if r.protocol != "tcp" {
			return nil, fmt.Errorf("only tcp protocol is allowed for proxy")
		}
		conn, err = libnet.DialTcpByProxy(r.proxyURL, r.proxyHost)
		if err != nil {
			return nil, err
		}
	} else {
		if r.addr == "" {
			r.addr = fmt.Sprintf("127.0.0.1:%d", r.port)
		}
		switch r.protocol {
		case "tcp":
			conn, err = net.Dial("tcp", r.addr)
		case "udp":
			conn, err = net.Dial("udp", r.addr)
		default:
			return nil, fmt.Errorf("invalid protocol")
		}
		if err != nil {
			return nil, err
		}
	}

	defer conn.Close()
	if r.timeout > 0 {
		conn.SetDeadline(time.Now().Add(r.timeout))
	}
	return sendRequestByConn(conn, r.body)
}

func SendTCPRequest(port int, content []byte, timeout time.Duration) ([]byte, error) {
	c, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return nil, fmt.Errorf("connect to tcp server error: %v", err)
	}
	defer c.Close()

	c.SetDeadline(time.Now().Add(timeout))
	return sendRequestByConn(c, content)
}

func SendUDPRequest(port int, content []byte, timeout time.Duration) ([]byte, error) {
	c, err := net.Dial("udp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return nil, fmt.Errorf("connect to udp server error:  %v", err)
	}
	defer c.Close()

	c.SetDeadline(time.Now().Add(timeout))
	return sendRequestByConn(c, content)
}

func sendRequestByConn(c net.Conn, content []byte) ([]byte, error) {
	_, err := c.Write(content)
	if err != nil {
		return nil, fmt.Errorf("write error: %v", err)
	}

	buf := make([]byte, 2048)
	n, err := c.Read(buf)
	if err != nil {
		return nil, fmt.Errorf("read error: %v", err)
	}
	return buf[:n], nil
}
