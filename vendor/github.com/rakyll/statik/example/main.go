package main

import (
	"log"
	"net/http"

	_ "github.com/rakyll/statik/example/statik"
	"github.com/rakyll/statik/fs"
)

// Before buildling, run `statik -src=./public`
// to generate the statik package.
// Then, run the main program and visit http://localhost:8080/public/hello.txt
func main() {
	statikFS, err := fs.New()
	if err != nil {
		log.Fatalf(err.Error())
	}

	http.Handle("/public/", http.StripPrefix("/public/", http.FileServer(statikFS)))
	http.ListenAndServe(":8080", nil)
}
