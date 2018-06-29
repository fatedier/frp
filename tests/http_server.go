package tests

import (
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{}

func StartHttpServer() {
	http.HandleFunc("/", handleHttp)
	http.HandleFunc("/ws", handleWebSocket)
	http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", TEST_HTTP_PORT), nil)
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

	if strings.Contains(r.Host, "127.0.0.1") || strings.Contains(r.Host, "test2.frp.com") ||
		strings.Contains(r.Host, "test5.frp.com") || strings.Contains(r.Host, "test6.frp.com") {
		w.WriteHeader(200)
		w.Write([]byte(TEST_HTTP_NORMAL_STR))
	} else if strings.Contains(r.Host, "test3.frp.com") {
		w.WriteHeader(200)
		if strings.Contains(r.URL.Path, "foo") {
			w.Write([]byte(TEST_HTTP_FOO_STR))
		} else if strings.Contains(r.URL.Path, "bar") {
			w.Write([]byte(TEST_HTTP_BAR_STR))
		} else {
			w.Write([]byte(TEST_HTTP_NORMAL_STR))
		}
	} else {
		w.WriteHeader(404)
	}
	return
}
