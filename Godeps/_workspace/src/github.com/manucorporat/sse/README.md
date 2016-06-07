#Server-Sent Events [![GoDoc](https://godoc.org/github.com/manucorporat/sse?status.svg)](https://godoc.org/github.com/manucorporat/sse) [![Build Status](https://travis-ci.org/manucorporat/sse.svg)](https://travis-ci.org/manucorporat/sse)

Server-sent events (SSE) is a technology where a browser receives automatic updates from a server via HTTP connection. The Server-Sent Events EventSource API is [standardized as part of HTML5[1] by the W3C](http://www.w3.org/TR/2009/WD-eventsource-20091029/).

- [Real world demostration using Gin](http://sse.getgin.io/)
- [Read this great SSE introduction by the HTML5Rocks guys](http://www.html5rocks.com/en/tutorials/eventsource/basics/)
- [Browser support](http://caniuse.com/#feat=eventsource)

##Sample code

```go
import "github.com/manucorporat/sse"

func httpHandler(w http.ResponseWriter, req *http.Request) {
	// data can be a primitive like a string, an integer or a float
	sse.Encode(w, sse.Event{
		Event: "message",
		Data:  "some data\nmore data",
	})

	// also a complex type, like a map, a struct or a slice
	sse.Encode(w, sse.Event{
		Id:    "124",
		Event: "message",
		Data: map[string]interface{}{
			"user":    "manu",
			"date":    time.Now().Unix(),
			"content": "hi!",
		},
	})
}
```
```
event: message
data: some data\\nmore data

id: 124
event: message
data: {"content":"hi!","date":1431540810,"user":"manu"}
 
```

##Content-Type

```go
fmt.Println(sse.ContentType)
```
```
text/event-stream
```

##Decoding support

There is a client-side implementation of SSE coming soon.