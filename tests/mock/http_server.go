package mock

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"regexp"
	"strings"

	"github.com/fatedier/frp/tests/consts"

	"github.com/gorilla/websocket"
)

type HttpServer struct {
	l net.Listener

	port    int
	handler http.HandlerFunc
}

func NewHttpServer(port int, handler http.HandlerFunc) *HttpServer {
	return &HttpServer{
		port:    port,
		handler: handler,
	}
}

func (hs *HttpServer) Start() error {
	l, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", hs.port))
	if err != nil {
		fmt.Printf("http server listen error: %v\n", err)
		return err
	}
	hs.l = l

	go http.Serve(l, http.HandlerFunc(hs.handler))
	return nil
}

func (hs *HttpServer) Stop() {
	hs.l.Close()
}

var upgrader = websocket.Upgrader{}

func StartHttpServer(port int) {
	http.HandleFunc("/", handleHttp)
	http.HandleFunc("/ws", handleWebSocket)
	http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", port), nil)
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer c.Close()
	for {
		mt, message, err := c.ReadMessage()
		if err != nil {
			break
		}
		err = c.WriteMessage(mt, message)
		if err != nil {
			log.Println("write:", err)
			break
		}
	}
}

func handleHttp(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("X-From-Where") == "frp" {
		w.Header().Set("X-Header-Set", "true")
	}

	match, err := regexp.Match(`.*\.sub\.com`, []byte(r.Host))
	if err != nil {
		w.WriteHeader(500)
		return
	}

	if match {
		w.WriteHeader(200)
		w.Write([]byte(r.Host))
		return
	}

	if strings.HasPrefix(r.Host, "127.0.0.1") || strings.HasPrefix(r.Host, "test2.frp.com") ||
		strings.HasPrefix(r.Host, "test5.frp.com") || strings.HasPrefix(r.Host, "test6.frp.com") ||
		strings.HasPrefix(r.Host, "test.frp1.com") || strings.HasPrefix(r.Host, "new.test.frp1.com") {

		w.WriteHeader(200)
		w.Write([]byte(consts.TEST_HTTP_NORMAL_STR))
	} else if strings.Contains(r.Host, "test3.frp.com") {
		w.WriteHeader(200)
		if strings.Contains(r.URL.Path, "foo") {
			w.Write([]byte(consts.TEST_HTTP_FOO_STR))
		} else if strings.Contains(r.URL.Path, "bar") {
			w.Write([]byte(consts.TEST_HTTP_BAR_STR))
		} else {
			w.Write([]byte(consts.TEST_HTTP_NORMAL_STR))
		}
	} else {
		w.WriteHeader(404)
	}
	return
}
