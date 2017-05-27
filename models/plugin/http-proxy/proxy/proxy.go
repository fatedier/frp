package proxy

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/fatedier/frp/models/plugin"
	"github.com/fatedier/frp/utils/log"
	wrap "github.com/fatedier/frp/utils/net"
)

var (
	HTTP_200  = []byte("HTTP/1.1 200 Connection Established\r\n\r\n")
	ProxyName = "default-proxy"
	cnfg      Config
)

func init() {
	// 加载配置文件
	err := cnfg.GetConfig("../config/config.json")
	if err != nil {
		log.Error("can not load config file:%v\n", err)
		os.Exit(-1)
	}
	plugin.Register(ProxyName, NewProxyPlugin)
}

type ProxyServer struct {
	Tr *http.Transport
}

type Proxy struct {
	Server *http.Server
	Ln     net.Listener
}

func NewProxyPlugin(params map[string]string) (p plugin.Plugin, err error) {

	listen, err := net.Listen("tcp", cnfg.Port)
	if err != nil {
		log.Error("can not listen %v port", cnfg.Port)
		return
	}

	proxy := &Proxy{
		Server: NewProxyServer(),
		Ln:     listen,
	}
	go proxy.Server.Serve(proxy.Ln)

	return proxy, nil
}

func (proxy *Proxy) Name() string {
	return ProxyName
}

// right??
func (proxy *Proxy) Handle(conn io.ReadWriteCloser) {
	wrapConn := wrap.WrapReadWriteCloserToConn(conn)

	remote, err := net.Dial("tcp", cnfg.Port)
	if err != nil {
		log.Error("dial tcp error:%v", err)
		return
	}

	// or tcp.Join(remote,wrapConn)
	_, err = io.Copy(remote, wrapConn)
	if err != nil && err != io.EOF {
		log.Error("io copy data error:%v", err)
		return
	}
	return
}

func (proxy *Proxy) Close() error {
	return proxy.Server.Close()
}

func NewProxyServer() *http.Server {

	return &http.Server{
		Addr:           cnfg.Port,
		Handler:        &ProxyServer{Tr: http.DefaultTransport.(*http.Transport)},
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
}

func (proxy *ProxyServer) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			log.Error("Panic: %v", err)
			fmt.Fprintf(rw, fmt.Sprintln(err))
		}
	}()

	if req.Method == "CONNECT" { // 是connect连接
		proxy.HttpsHandler(rw, req)
	} else {
		proxy.HttpHandler(rw, req)
	}
}

// 处理普通的http请求
func (proxy *ProxyServer) HttpHandler(rw http.ResponseWriter, req *http.Request) {
	log.Info("is sending request %v %v ", req.Method, req.URL.Host)
	removeProxyHeaders(req) // 去除不必要的头

	resp, err := proxy.Tr.RoundTrip(req)
	if err != nil {
		log.Error("transport RoundTrip error: %v", err)
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	clearHeaders(rw.Header()) // 得到一个空的Header
	copyHeaders(rw.Header(), resp.Header)
	rw.WriteHeader(resp.StatusCode)

	nr, err := io.Copy(rw, resp.Body)
	if err != nil && err != io.EOF {
		log.Error("got an error when copy remote response to client.%v", err)
		return
	}
	log.Info("copied %v bytes from remote host %v.", nr, req.URL.Host)
}

// 处理https连接，主要用于CONNECT方法
func (proxy *ProxyServer) HttpsHandler(rw http.ResponseWriter, req *http.Request) {
	log.Info("[CONNECT] tried to connect to remote host %v", req.URL.Host)

	hj, _ := rw.(http.Hijacker)
	client, _, err := hj.Hijack() //获取客户端与代理服务器的tcp连接
	if err != nil {
		log.Error("failed to get Tcp connection of", req.RequestURI)
		http.Error(rw, "Failed", http.StatusBadRequest)
		return
	}

	remote, err := net.Dial("tcp", req.URL.Host) //建立服务端和代理服务器的tcp连接
	if err != nil {
		log.Error("failed to connect %v", req.RequestURI)
		http.Error(rw, "Failed", http.StatusBadRequest)
		client.Close()
		return
	}

	client.Write(HTTP_200)

	go copyRemoteToClient(remote, client)
	go copyRemoteToClient(client, remote)
}

// data copy between two socket
func copyRemoteToClient(remote, client net.Conn) {
	defer func() {
		remote.Close()
		client.Close()
	}()

	nr, err := io.Copy(remote, client)
	if err != nil && err != io.EOF {
		log.Error("got an error when handles CONNECT %v", err)
		return
	}
	log.Info("[CONNECT]  transported %v bytes betwwen %v and %v", nr, remote.RemoteAddr(), client.RemoteAddr())
}

func copyHeaders(dst, src http.Header) {
	for key, values := range src {
		for _, value := range values {
			dst.Add(key, value)
		}
	}
}

func clearHeaders(headers http.Header) {
	for key, _ := range headers {
		headers.Del(key)
	}
}

func removeProxyHeaders(req *http.Request) {
	req.RequestURI = ""
	req.Header.Del("Proxy-Connection")
	req.Header.Del("Connection")
	req.Header.Del("Keep-Alive")
	req.Header.Del("Proxy-Authenticate")
	req.Header.Del("Proxy-Authorization")
	req.Header.Del("TE")
	req.Header.Del("Trailers")
	req.Header.Del("Transfer-Encoding")
	req.Header.Del("Upgrade")
}
