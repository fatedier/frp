package echoserver

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"

	fnet "github.com/fatedier/frp/pkg/util/net"

	"github.com/gorilla/websocket"
)

type ServerType string

const (
	TCP   ServerType = "tcp"
	UDP   ServerType = "udp"
	Unix  ServerType = "unix"
	HTTP  ServerType = "http"
	HTTPS ServerType = "https"
	WS    ServerType = "ws"
	WSS   ServerType = "wss"
)

type Options struct {
	Type              ServerType
	BindAddr          string
	BindPort          int32
	RepeatNum         int
	SpecifiedResponse string
}

type Server struct {
	opt Options

	l           net.Listener
	httpServer  *http.Server
	httpsServer *http.Server
	wsServer    *http.Server
	wssServer   *http.Server
}

func New(opt Options) *Server {
	if opt.Type == "" {
		opt.Type = TCP
	}
	if opt.BindAddr == "" {
		opt.BindAddr = "127.0.0.1"
	}
	if opt.RepeatNum <= 0 {
		opt.RepeatNum = 1
	}
	return &Server{
		opt: opt,
	}
}

func (s *Server) GetOptions() Options {
	return s.opt
}

func (s *Server) Run() error {
	if err := s.init(); err != nil {
		return err
	}

	switch s.opt.Type {
	case TCP, UDP, Unix:
		go func() {
			for {
				c, err := s.l.Accept()
				if err != nil {
					return
				}
				go s.handle(c)
			}
		}()
	case HTTP:
		go func() {
			err := s.httpServer.ListenAndServe()
			if err != nil {
				fmt.Printf("echo http server exited error: %v\n", err)
			}
		}()
	case HTTPS:
		go func() {
			err := s.httpsServer.ListenAndServeTLS("e2e.crt", "e2e.key")
			if err != nil {
				fmt.Printf("echo https server exited error: %v\n", err)
			}
		}()
	case WS:
		go func() {
			err := s.wsServer.ListenAndServe()
			if err != nil {
				fmt.Printf("echo ws server exited error: %v\n", err)
			}
		}()
	case WSS:
		go func() {
			err := s.wssServer.ListenAndServeTLS("e2e.crt", "e2e.key")
			if err != nil {
				fmt.Printf("echo wss server exited error: %v\n", err)
			}
		}()
	default:
		return fmt.Errorf("unknown server type: %s", s.opt.Type)
	}

	return nil
}

func (s *Server) Close() error {
	if s.l != nil {
		return s.l.Close()
	}
	if s.httpServer != nil {
		s.httpServer.Close()
	}
	if s.httpsServer != nil {
		s.httpsServer.Close()
	}
	if s.wsServer != nil {
		s.wsServer.Close()
	}
	if s.wssServer != nil {
		s.wssServer.Close()
	}
	return nil
}

func (s *Server) init() (err error) {
	switch s.opt.Type {
	case TCP:
		s.l, err = net.Listen("tcp", fmt.Sprintf("%s:%d", s.opt.BindAddr, s.opt.BindPort))
	case UDP:
		s.l, err = fnet.ListenUDP(s.opt.BindAddr, int(s.opt.BindPort))
	case Unix:
		s.l, err = net.Listen("unix", s.opt.BindAddr)
	case HTTP:
		s.httpServer = &http.Server{
			Addr:    fmt.Sprintf("%s:%d", s.opt.BindAddr, s.opt.BindPort),
			Handler: http.HandlerFunc(echo),
		}
		fmt.Println("http echo server listen port", s.opt.BindPort)
	case HTTPS:
		s.httpsServer = &http.Server{
			Addr:    fmt.Sprintf("%s:%d", s.opt.BindAddr, s.opt.BindPort),
			Handler: http.HandlerFunc(echo),
		}
		fmt.Println("https echo server listen port", s.opt.BindPort)
	case WS:
		s.wsServer = &http.Server{
			Addr:    fmt.Sprintf("%s:%d", s.opt.BindAddr, s.opt.BindPort),
			Handler: http.HandlerFunc(ws),
		}
		fmt.Println("ws echo server listen port", s.opt.BindPort)
	case WSS:
		s.wssServer = &http.Server{
			Addr:    fmt.Sprintf("%s:%d", s.opt.BindAddr, s.opt.BindPort),
			Handler: http.HandlerFunc(ws),
		}
		fmt.Println("wss echo server listen port", s.opt.BindPort)
	default:
		return fmt.Errorf("unknown server type: %s", s.opt.Type)
	}
	if err != nil {
		return
	}
	return nil
}

func (s *Server) handle(c net.Conn) {
	defer c.Close()

	buf := make([]byte, 2048)
	for {
		n, err := c.Read(buf)
		if err != nil {
			return
		}

		var response string
		if len(s.opt.SpecifiedResponse) > 0 {
			response = s.opt.SpecifiedResponse
		} else {
			response = strings.Repeat(string(buf[:n]), s.opt.RepeatNum)
		}
		c.Write([]byte(response))
	}
}

func echo(w http.ResponseWriter, r *http.Request) {
	rh := make(map[string]string)
	for key, _ := range r.Header {
		rh[key] = r.Header.Get(key)
	}
	rh["Host"] = r.Host
	rh["Path"] = r.URL.Path

	data, err := json.Marshal(rh)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(data)
	return
}

var (
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
)

func ws(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		if _, ok := err.(websocket.HandshakeError); !ok {
			fmt.Println(err)
		}
		return
	}

	go func() {
		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				fmt.Printf("reading message error: %v\n", err)
			}
			if err = conn.WriteMessage(websocket.TextMessage, data); err != nil {
				fmt.Printf("writing message error: %v\n", err)
			}
		}
	}()
}
