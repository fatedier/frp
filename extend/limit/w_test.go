package limit

import (
	"fmt"
	"net/http"
	"testing"
)

func TestHttp(t *testing.T) {
	http.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		lw := NewWriterWithLimit(w, 10*KB)
		for {
			fmt.Fprintf(lw, "x")
		}
	})
	http.ListenAndServe(":62542", nil)
}
