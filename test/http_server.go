package main

import (
	"fmt"
	"net/http"
)

var (
	PORT         int64  = 10702
	HTTP_RES_STR string = "Hello World"
)

func main() {
	http.HandleFunc("/", request)
	http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", PORT), nil)
}

func request(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(HTTP_RES_STR))
}
