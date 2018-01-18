package tests

import (
	"fmt"
	"net/http"
	"strings"
)

func StartHttpServer() {
	http.HandleFunc("/", request)
	http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", TEST_HTTP_PORT), nil)
}

func request(w http.ResponseWriter, r *http.Request) {
	if strings.Contains(r.Host, "127.0.0.1") || strings.Contains(r.Host, "test2.frp.com") ||
		strings.Contains(r.Host, "test5.frp.com") {
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
