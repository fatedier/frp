package test

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
	"time"

	"frp/utils/conn"
)

var (
	ECHO_PORT     int64  = 10711
	HTTP_PORT     int64  = 10710
	ECHO_TEST_STR string = "Hello World\n"
	HTTP_RES_STR  string = "Hello World"
)

func TestEchoServer(t *testing.T) {
	c, err := conn.ConnectServer("0.0.0.0", ECHO_PORT)
	if err != nil {
		t.Fatalf("connect to echo server error: %v", err)
	}
	timer := time.Now().Add(time.Duration(5) * time.Second)
	c.SetDeadline(timer)

	c.Write(ECHO_TEST_STR)

	buff, err := c.ReadLine()
	if err != nil {
		t.Fatalf("read from echo server error: %v", err)
	}

	if ECHO_TEST_STR != buff {
		t.Fatalf("content error, send [%s], get [%s]", strings.Trim(ECHO_TEST_STR, "\n"), strings.Trim(buff, "\n"))
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
