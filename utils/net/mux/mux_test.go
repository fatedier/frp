package mux

import (
	"bufio"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func runHttpSvr(ln net.Listener) *httptest.Server {
	svr := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("http service"))
	}))
	svr.Listener = ln
	svr.Start()
	return svr
}

func runHttpsSvr(ln net.Listener) *httptest.Server {
	svr := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("https service"))
	}))
	svr.Listener = ln
	svr.StartTLS()
	return svr
}

func runEchoSvr(ln net.Listener) {
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			rd := bufio.NewReader(conn)
			data, err := rd.ReadString('\n')
			if err != nil {
				return
			}
			conn.Write([]byte(data))
			conn.Close()
		}
	}()
}

func TestMux(t *testing.T) {
	assert := assert.New(t)

	ln, err := net.Listen("tcp", "127.0.0.1:")
	assert.NoError(err)

	mux := NewMux()
	httpLn := mux.ListenHttp(0)
	httpsLn := mux.ListenHttps(0)
	defaultLn := mux.DefaultListener()
	go mux.Serve(ln)
	time.Sleep(100 * time.Millisecond)

	httpSvr := runHttpSvr(httpLn)
	defer httpSvr.Close()
	httpsSvr := runHttpsSvr(httpsLn)
	defer httpsSvr.Close()
	runEchoSvr(defaultLn)
	defer ln.Close()

	// test http service
	resp, err := http.Get(httpSvr.URL)
	assert.NoError(err)
	data, err := ioutil.ReadAll(resp.Body)
	assert.NoError(err)
	assert.Equal("http service", string(data))

	// test https service
	client := httpsSvr.Client()
	resp, err = client.Get(httpsSvr.URL)
	assert.NoError(err)
	data, err = ioutil.ReadAll(resp.Body)
	assert.NoError(err)
	assert.Equal("https service", string(data))

	// test echo service
	conn, err := net.Dial("tcp", ln.Addr().String())
	assert.NoError(err)
	_, err = conn.Write([]byte("test echo\n"))
	assert.NoError(err)
	data = make([]byte, 1024)
	n, err := conn.Read(data)
	assert.NoError(err)
	assert.Equal("test echo\n", string(data[:n]))
}
