package tests

import (
	"fmt"
	"net/http"
)

func StartHttpServer() {
	http.HandleFunc("/", request)
	http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", 10702), nil)
}

func request(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(HTTP_RES_STR))
}
