package proxy

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
	"util"

	"github.com/fatedier/frp/utils/log"
)

var HTTP_200 = []byte("HTTP/1.1 200 Connection Established\r\n\r\n")

type ProxyServer struct {
	Tr   *http.Transport
	Name string
}

func NewProxyServer() *http.Server {

	return &http.Server{
		Addr:           cnfg.Port,
		Handler:        &ProxyServer{Tr: http.DefaultTransport.(*http.Transport), Name: "default-proxy"},
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
}

// 自动执行的方法，因为ProxyServer实现了Handler接口,需要ServerHTTP
func (proxy *ProxyServer) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			log.Error("Panic: %v", err)
			fmt.Fprintf(rw, fmt.Sprintln(err))
		}
	}()

	// 鉴权
	if proxy.Auth(rw, req) {
		return
	}

	if req.Method == "CONNECT" { // 是connect连接
		proxy.HttpsHandler(rw, req)
	} else {
		proxy.HttpHandler(rw, req)
	}
}

// 处理普通的http请求
func (proxy *ProxyServer) HttpHandler(rw http.ResponseWriter, req *http.Request) {
	log.Info("%v is sending request %v %v ", proxy.Name, req.Method, req.URL.Host)
	util.RemoveProxyHeaders(req) // 去除不必要的头

	resp, err := proxy.Tr.RoundTrip(req)
	if err != nil {
		log.Error("%s transport RoundTrip error: %v", proxy.Name, err)
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	util.ClearHeaders(rw.Header()) // 得到一个空的Header
	util.CopyHeaders(rw.Header(), resp.Header)
	rw.WriteHeader(resp.StatusCode)

	nr, err := io.Copy(rw, resp.Body)
	if err != nil && err != io.EOF {
		log.Error("%v got an error when copy remote response to client.%v", proxy.Name, err)
		return
	}
	log.Info("%v copied %v bytes from remote host %v.", proxy.Name, nr, req.URL.Host)
}

// 处理https连接，主要用于CONNECT方法
func (proxy *ProxyServer) HttpsHandler(rw http.ResponseWriter, req *http.Request) {
	log.Info("[CONNECT] %v tried to connect to remote host %v", proxy.Name, req.URL.Host)

	hj, _ := rw.(http.Hijacker)
	client, _, err := hj.Hijack() //获取客户端与代理服务器的tcp连接
	if err != nil {
		log.Error("%v failed to get Tcp connection of", proxy.Name, req.RequestURI)
		http.Error(rw, "Failed", http.StatusBadRequest)
		return
	}

	remote, err := net.Dial("tcp", req.URL.Host) //建立服务端和代理服务器的tcp连接
	if err != nil {
		log.Error("%v failed to connect %v", proxy.Name, req.RequestURI)
		http.Error(rw, "Failed", http.StatusBadRequest)
		client.Close()
		return
	}

	client.Write(HTTP_200)

	go copyRemoteToClient(proxy.Name, remote, client)
	go copyRemoteToClient(proxy.Name, client, remote)
}

// data copy between two socket
func copyRemoteToClient(Name string, remote, client net.Conn) {
	defer func() {
		remote.Close()
		client.Close()
	}()

	nr, err := io.Copy(remote, client)
	if err != nil && err != io.EOF {
		log.Error("%v got an error when handles CONNECT %v", Name, err)
		return
	}
	log.Info("[CONNECT] %v transported %v bytes betwwen %v and %v", Name, nr, remote.RemoteAddr(), client.RemoteAddr())
}
