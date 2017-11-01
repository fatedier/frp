package main

import (
	"archive/zip"
	_ "bytes"
	"fmt"
	"golang.org/x/net/html"
	"io"
	"net/http"
	"os"
	"strings"
)

// Download all CPUID dumps from http://users.atw.hu/instlatx64/
func main() {
	resp, err := http.Get("http://users.atw.hu/instlatx64/?")
	if err != nil {
		panic(err)
	}

	node, err := html.Parse(resp.Body)
	if err != nil {
		panic(err)
	}

	file, err := os.Create("cpuid_data.zip")
	if err != nil {
		panic(err)
	}
	defer file.Close()
	gw := zip.NewWriter(file)

	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, a := range n.Attr {
				if a.Key == "href" {
					err := ParseURL(a.Val, gw)
					if err != nil {
						panic(err)
					}
					break
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}

	f(node)
	err = gw.Close()
	if err != nil {
		panic(err)
	}
}

func ParseURL(s string, gw *zip.Writer) error {
	if strings.Contains(s, "CPUID.txt") {
		fmt.Println("Adding", "http://users.atw.hu/instlatx64/"+s)
		resp, err := http.Get("http://users.atw.hu/instlatx64/" + s)
		if err != nil {
			fmt.Println("Error getting ", s, ":", err)
		}
		defer resp.Body.Close()
		w, err := gw.Create(s)
		if err != nil {
			return err
		}

		_, err = io.Copy(w, resp.Body)
		if err != nil {
			return err
		}
	}
	return nil
}
