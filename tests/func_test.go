package tests

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	frpNet "github.com/fatedier/frp/utils/net"
)

var (
	ECHO_PORT     int64  = 10711
	UDP_ECHO_PORT int64  = 10712
	HTTP_PORT     int64  = 10710
	ECHO_TEST_STR string = "Hello World\n"
	HTTP_RES_STR  string = "Hello World"
)

func init() {
	go StartEchoServer()
	go StartUdpEchoServer()
	go StartHttpServer()
	go StartUnixDomainServer()
	time.Sleep(500 * time.Millisecond)
}

func TestEchoServer(t *testing.T) {
	c, err := frpNet.ConnectTcpServer(fmt.Sprintf("127.0.0.1:%d", ECHO_PORT))
	if err != nil {
		t.Fatalf("connect to echo server error: %v", err)
	}
	timer := time.Now().Add(time.Duration(5) * time.Second)
	c.SetDeadline(timer)

	c.Write([]byte(ECHO_TEST_STR + "\n"))

	br := bufio.NewReader(c)
	buf, err := br.ReadString('\n')
	if err != nil {
		t.Fatalf("read from echo server error: %v", err)
	}

	if ECHO_TEST_STR != buf {
		t.Fatalf("content error, send [%s], get [%s]", strings.Trim(ECHO_TEST_STR, "\n"), strings.Trim(buf, "\n"))
	}
}

func TestHttpServer(t *testing.T) {
	client := &http.Client{}
	req, _ := http.NewRequest("GET", fmt.Sprintf("http://127.0.0.1:%d", HTTP_PORT), nil)
	res, err := client.Do(req)
	if err != nil {
		t.Fatalf("do http request error: %v", err)
	}
	if res.StatusCode == 200 {
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			t.Fatalf("read from http server error: %v", err)
		}
		bodystr := string(body)
		if bodystr != HTTP_RES_STR {
			t.Fatalf("content from http server error [%s], correct string is [%s]", bodystr, HTTP_RES_STR)
		}
	} else {
		t.Fatalf("http code from http server error [%d]", res.StatusCode)
	}
}

func TestUdpEchoServer(t *testing.T) {
	addr, err := net.ResolveUDPAddr("udp", "127.0.0.1:10712")
	if err != nil {
		t.Fatalf("do udp request error: %v", err)
	}
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		t.Fatalf("dial udp server error: %v", err)
	}
	defer conn.Close()
	_, err = conn.Write([]byte("hello frp\n"))
	if err != nil {
		t.Fatalf("write to udp server error: %v", err)
	}
	data := make([]byte, 20)
	n, err := conn.Read(data)
	if err != nil {
		t.Fatalf("read from udp server error: %v", err)
	}

	if string(bytes.TrimSpace(data[:n])) != "hello frp" {
		t.Fatalf("message got from udp server error, get %s", string(data[:n-1]))
	}
}

func TestUnixDomainServer(t *testing.T) {
	c, err := frpNet.ConnectTcpServer(fmt.Sprintf("127.0.0.1:%d", 10704))
	if err != nil {
		t.Fatalf("connect to echo server error: %v", err)
	}
	timer := time.Now().Add(time.Duration(5) * time.Second)
	c.SetDeadline(timer)

	c.Write([]byte(ECHO_TEST_STR + "\n"))

	br := bufio.NewReader(c)
	buf, err := br.ReadString('\n')
	if err != nil {
		t.Fatalf("read from echo server error: %v", err)
	}

	if ECHO_TEST_STR != buf {
		t.Fatalf("content error, send [%s], get [%s]", strings.Trim(ECHO_TEST_STR, "\n"), strings.Trim(buf, "\n"))
	}
}
