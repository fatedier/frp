# gorilla/mux

[![GoDoc](https://godoc.org/github.com/gorilla/mux?status.svg)](https://godoc.org/github.com/gorilla/mux)
[![Build Status](https://travis-ci.org/gorilla/mux.svg?branch=master)](https://travis-ci.org/gorilla/mux)
[![Sourcegraph](https://sourcegraph.com/github.com/gorilla/mux/-/badge.svg)](https://sourcegraph.com/github.com/gorilla/mux?badge)

![Gorilla Logo](http://www.gorillatoolkit.org/static/images/gorilla-icon-64.png)

http://www.gorillatoolkit.org/pkg/mux

Package `gorilla/mux` implements a request router and dispatcher for matching incoming requests to
their respective handler.

The name mux stands for "HTTP request multiplexer". Like the standard `http.ServeMux`, `mux.Router` matches incoming requests against a list of registered routes and calls a handler for the route that matches the URL or other conditions. The main features are:

* It implements the `http.Handler` interface so it is compatible with the standard `http.ServeMux`.
* Requests can be matched based on URL host, path, path prefix, schemes, header and query values, HTTP methods or using custom matchers.
* URL hosts, paths and query values can have variables with an optional regular expression.
* Registered URLs can be built, or "reversed", which helps maintaining references to resources.
* Routes can be used as subrouters: nested routes are only tested if the parent route matches. This is useful to define groups of routes that share common conditions like a host, a path prefix or other repeated attributes. As a bonus, this optimizes request matching.

---

* [Install](#install)
* [Examples](#examples)
* [Matching Routes](#matching-routes)
* [Static Files](#static-files)
* [Registered URLs](#registered-urls)
* [Walking Routes](#walking-routes)
* [Graceful Shutdown](#graceful-shutdown)
* [Middleware](#middleware)
* [Testing Handlers](#testing-handlers)
* [Full Example](#full-example)

---

## Install

With a [correctly configured](https://golang.org/doc/install#testing) Go toolchain:

```sh
go get -u github.com/gorilla/mux
```

## Examples

Let's start registering a couple of URL paths and handlers:

```go
func main() {
    r := mux.NewRouter()
    r.HandleFunc("/", HomeHandler)
    r.HandleFunc("/products", ProductsHandler)
    r.HandleFunc("/articles", ArticlesHandler)
    http.Handle("/", r)
}
```

Here we register three routes mapping URL paths to handlers. This is equivalent to how `http.HandleFunc()` works: if an incoming request URL matches one of the paths, the corresponding handler is called passing (`http.ResponseWriter`, `*http.Request`) as parameters.

Paths can have variables. They are defined using the format `{name}` or `{name:pattern}`. If a regular expression pattern is not defined, the matched variable will be anything until the next slash. For example:

```go
r := mux.NewRouter()
r.HandleFunc("/products/{key}", ProductHandler)
r.HandleFunc("/articles/{category}/", ArticlesCategoryHandler)
r.HandleFunc("/articles/{category}/{id:[0-9]+}", ArticleHandler)
```

The names are used to create a map of route variables which can be retrieved calling `mux.Vars()`:

```go
func ArticlesCategoryHandler(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    w.WriteHeader(http.StatusOK)
    fmt.Fprintf(w, "Category: %v\n", vars["category"])
}
```

And this is all you need to know about the basic usage. More advanced options are explained below.

### Matching Routes

Routes can also be restricted to a domain or subdomain. Just define a host pattern to be matched. They can also have variables:

```go
r := mux.NewRouter()
// Only matches if domain is "www.example.com".
r.Host("www.example.com")
// Matches a dynamic subdomain.
r.Host("{subdomain:[a-z]+}.domain.com")
```

There are several other matchers that can be added. To match path prefixes:

```go
r.PathPrefix("/products/")
```

...or HTTP methods:

```go
r.Methods("GET", "POST")
```

...or URL schemes:

```go
r.Schemes("https")
```

...or header values:

```go
r.Headers("X-Requested-With", "XMLHttpRequest")
```

...or query values:

```go
r.Queries("key", "value")
```

...or to use a custom matcher function:

```go
r.MatcherFunc(func(r *http.Request, rm *RouteMatch) bool {
    return r.ProtoMajor == 0
})
```

...and finally, it is possible to combine several matchers in a single route:

```go
r.HandleFunc("/products", ProductsHandler).
  Host("www.example.com").
  Methods("GET").
  Schemes("http")
```

Routes are tested in the order they were added to the router. If two routes match, the first one wins:

```go
r := mux.NewRouter()
r.HandleFunc("/specific", specificHandler)
r.PathPrefix("/").Handler(catchAllHandler)
```

Setting the same matching conditions again and again can be boring, so we have a way to group several routes that share the same requirements. We call it "subrouting".

For example, let's say we have several URLs that should only match when the host is `www.example.com`. Create a route for that host and get a "subrouter" from it:

```go
r := mux.NewRouter()
s := r.Host("www.example.com").Subrouter()
```

Then register routes in the subrouter:

```go
s.HandleFunc("/products/", ProductsHandler)
s.HandleFunc("/products/{key}", ProductHandler)
s.HandleFunc("/articles/{category}/{id:[0-9]+}", ArticleHandler)
```

The three URL paths we registered above will only be tested if the domain is `www.example.com`, because the subrouter is tested first. This is not only convenient, but also optimizes request matching. You can create subrouters combining any attribute matchers accepted by a route.

Subrouters can be used to create domain or path "namespaces": you define subrouters in a central place and then parts of the app can register its paths relatively to a given subrouter.

There's one more thing about subroutes. When a subrouter has a path prefix, the inner routes use it as base for their paths:

```go
r := mux.NewRouter()
s := r.PathPrefix("/products").Subrouter()
// "/products/"
s.HandleFunc("/", ProductsHandler)
// "/products/{key}/"
s.HandleFunc("/{key}/", ProductHandler)
// "/products/{key}/details"
s.HandleFunc("/{key}/details", ProductDetailsHandler)
```


### Static Files

Note that the path provided to `PathPrefix()` represents a "wildcard": calling
`PathPrefix("/static/").Handler(...)` means that the handler will be passed any
request that matches "/static/\*". This makes it easy to serve static files with mux:

```go
func main() {
    var dir string

    flag.StringVar(&dir, "dir", ".", "the directory to serve files from. Defaults to the current dir")
    flag.Parse()
    r := mux.NewRouter()

    // This will serve files under http://localhost:8000/static/<filename>
    r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir(dir))))

    srv := &http.Server{
        Handler:      r,
        Addr:         "127.0.0.1:8000",
        // Good practice: enforce timeouts for servers you create!
        WriteTimeout: 15 * time.Second,
        ReadTimeout:  15 * time.Second,
    }

    log.Fatal(srv.ListenAndServe())
}
```

### Registered URLs

Now let's see how to build registered URLs.

Routes can be named. All routes that define a name can have their URLs built, or "reversed". We define a name calling `Name()` on a route. For example:

```go
r := mux.NewRouter()
r.HandleFunc("/articles/{category}/{id:[0-9]+}", ArticleHandler).
  Name("article")
```

To build a URL, get the route and call the `URL()` method, passing a sequence of key/value pairs for the route variables. For the previous route, we would do:

```go
url, err := r.Get("article").URL("category", "technology", "id", "42")
```

...and the result will be a `url.URL` with the following path:

```
"/articles/technology/42"
```

This also works for host and query value variables:

```go
r := mux.NewRouter()
r.Host("{subdomain}.domain.com").
  Path("/articles/{category}/{id:[0-9]+}").
  Queries("filter", "{filter}").
  HandlerFunc(ArticleHandler).
  Name("article")

// url.String() will be "http://news.domain.com/articles/technology/42?filter=gorilla"
url, err := r.Get("article").URL("subdomain", "news",
                                 "category", "technology",
                                 "id", "42",
                                 "filter", "gorilla")
```

All variables defined in the route are required, and their values must conform to the corresponding patterns. These requirements guarantee that a generated URL will always match a registered route -- the only exception is for explicitly defined "build-only" routes which never match.

Regex support also exists for matching Headers within a route. For example, we could do:

```go
r.HeadersRegexp("Content-Type", "application/(text|json)")
```

...and the route will match both requests with a Content-Type of `application/json` as well as `application/text`

There's also a way to build only the URL host or path for a route: use the methods `URLHost()` or `URLPath()` instead. For the previous route, we would do:

```go
// "http://news.domain.com/"
host, err := r.Get("article").URLHost("subdomain", "news")

// "/articles/technology/42"
path, err := r.Get("article").URLPath("category", "technology", "id", "42")
```

And if you use subrouters, host and path defined separately can be built as well:

```go
r := mux.NewRouter()
s := r.Host("{subdomain}.domain.com").Subrouter()
s.Path("/articles/{category}/{id:[0-9]+}").
  HandlerFunc(ArticleHandler).
  Name("article")

// "http://news.domain.com/articles/technology/42"
url, err := r.Get("article").URL("subdomain", "news",
                                 "category", "technology",
                                 "id", "42")
```

### Walking Routes

The `Walk` function on `mux.Router` can be used to visit all of the routes that are registered on a router. For example,
the following prints all of the registered routes:

```go
package main

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

func handler(w http.ResponseWriter, r *http.Request) {
	return
}

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/", handler)
	r.HandleFunc("/products", handler).Methods("POST")
	r.HandleFunc("/articles", handler).Methods("GET")
	r.HandleFunc("/articles/{id}", handler).Methods("GET", "PUT")
	r.HandleFunc("/authors", handler).Queries("surname", "{surname}")
	err := r.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		pathTemplate, err := route.GetPathTemplate()
		if err == nil {
			fmt.Println("ROUTE:", pathTemplate)
		}
		pathRegexp, err := route.GetPathRegexp()
		if err == nil {
			fmt.Println("Path regexp:", pathRegexp)
		}
		queriesTemplates, err := route.GetQueriesTemplates()
		if err == nil {
			fmt.Println("Queries templates:", strings.Join(queriesTemplates, ","))
		}
		queriesRegexps, err := route.GetQueriesRegexp()
		if err == nil {
			fmt.Println("Queries regexps:", strings.Join(queriesRegexps, ","))
		}
		methods, err := route.GetMethods()
		if err == nil {
			fmt.Println("Methods:", strings.Join(methods, ","))
		}
		fmt.Println()
		return nil
	})

	if err != nil {
		fmt.Println(err)
	}

	http.Handle("/", r)
}
```

### Graceful Shutdown

Go 1.8 introduced the ability to [gracefully shutdown](https://golang.org/doc/go1.8#http_shutdown) a `*http.Server`. Here's how to do that alongside `mux`:

```go
package main

import (
    "context"
    "flag"
    "log"
    "net/http"
    "os"
    "os/signal"
    "time"

    "github.com/gorilla/mux"
)

func main() {
    var wait time.Duration
    flag.DurationVar(&wait, "graceful-timeout", time.Second * 15, "the duration for which the server gracefully wait for existing connections to finish - e.g. 15s or 1m")
    flag.Parse()

    r := mux.NewRouter()
    // Add your routes as needed

    srv := &http.Server{
        Addr:         "0.0.0.0:8080",
        // Good practice to set timeouts to avoid Slowloris attacks.
        WriteTimeout: time.Second * 15,
        ReadTimeout:  time.Second * 15,
        IdleTimeout:  time.Second * 60,
        Handler: r, // Pass our instance of gorilla/mux in.
    }

    // Run our server in a goroutine so that it doesn't block.
    go func() {
        if err := srv.ListenAndServe(); err != nil {
            log.Println(err)
        }
    }()

    c := make(chan os.Signal, 1)
    // We'll accept graceful shutdowns when quit via SIGINT (Ctrl+C)
    // SIGKILL, SIGQUIT or SIGTERM (Ctrl+/) will not be caught.
    signal.Notify(c, os.Interrupt)

    // Block until we receive our signal.
    <-c

    // Create a deadline to wait for.
    ctx, cancel := context.WithTimeout(context.Background(), wait)
    defer cancel()
    // Doesn't block if no connections, but will otherwise wait
    // until the timeout deadline.
    srv.Shutdown(ctx)
    // Optionally, you could run srv.Shutdown in a goroutine and block on
    // <-ctx.Done() if your application should wait for other services
    // to finalize based on context cancellation.
    log.Println("shutting down")
    os.Exit(0)
}
```

### Middleware

Mux supports the addition of middlewares to a [Router](https://godoc.org/github.com/gorilla/mux#Router), which are executed in the order they are added if a match is found, including its subrouters.
Middlewares are (typically) small pieces of code which take one request, do something with it, and pass it down to another middleware or the final handler. Some common use cases for middleware are request logging, header manipulation, or `ResponseWriter` hijacking.

Mux middlewares are defined using the de facto standard type:

```go
type MiddlewareFunc func(http.Handler) http.Handler
```

Typically, the returned handler is a closure which does something with the http.ResponseWriter and http.Request passed to it, and then calls the handler passed as parameter to the MiddlewareFunc. This takes advantage of closures being able access variables from the context where they are created, while retaining the signature enforced by the receivers.

A very basic middleware which logs the URI of the request being handled could be written as:

```go
func loggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Do stuff here
        log.Println(r.RequestURI)
        // Call the next handler, which can be another middleware in the chain, or the final handler.
        next.ServeHTTP(w, r)
    })
}
```

Middlewares can be added to a router using `Router.Use()`:

```go
r := mux.NewRouter()
r.HandleFunc("/", handler)
r.Use(loggingMiddleware)
```

A more complex authentication middleware, which maps session token to users, could be written as:

```go
// Define our struct
type authenticationMiddleware struct {
	tokenUsers map[string]string
}

// Initialize it somewhere
func (amw *authenticationMiddleware) Populate() {
	amw.tokenUsers["00000000"] = "user0"
	amw.tokenUsers["aaaaaaaa"] = "userA"
	amw.tokenUsers["05f717e5"] = "randomUser"
	amw.tokenUsers["deadbeef"] = "user0"
}

// Middleware function, which will be called for each request
func (amw *authenticationMiddleware) Middleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        token := r.Header.Get("X-Session-Token")

        if user, found := amw.tokenUsers[token]; found {
        	// We found the token in our map
        	log.Printf("Authenticated user %s\n", user)
        	// Pass down the request to the next middleware (or final handler)
        	next.ServeHTTP(w, r)
        } else {
        	// Write an error and stop the handler chain
        	http.Error(w, "Forbidden", http.StatusForbidden)
        }
    })
}
```

```go
r := mux.NewRouter()
r.HandleFunc("/", handler)

amw := authenticationMiddleware{}
amw.Populate()

r.Use(amw.Middleware)
```

Note: The handler chain will be stopped if your middleware doesn't call `next.ServeHTTP()` with the corresponding parameters. This can be used to abort a request if the middleware writer wants to. Middlewares _should_ write to `ResponseWriter` if they _are_ going to terminate the request, and they _should not_ write to `ResponseWriter` if they _are not_ going to terminate it.

### Testing Handlers

Testing handlers in a Go web application is straightforward, and _mux_ doesn't complicate this any further. Given two files: `endpoints.go` and `endpoints_test.go`, here's how we'd test an application using _mux_.

First, our simple HTTP handler:

```go
// endpoints.go
package main

func HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
    // A very simple health check.
    w.WriteHeader(http.StatusOK)
    w.Header().Set("Content-Type", "application/json")

    // In the future we could report back on the status of our DB, or our cache
    // (e.g. Redis) by performing a simple PING, and include them in the response.
    io.WriteString(w, `{"alive": true}`)
}

func main() {
    r := mux.NewRouter()
    r.HandleFunc("/health", HealthCheckHandler)

    log.Fatal(http.ListenAndServe("localhost:8080", r))
}
```

Our test code:

```go
// endpoints_test.go
package main

import (
    "net/http"
    "net/http/httptest"
    "testing"
)

func TestHealthCheckHandler(t *testing.T) {
    // Create a request to pass to our handler. We don't have any query parameters for now, so we'll
    // pass 'nil' as the third parameter.
    req, err := http.NewRequest("GET", "/health", nil)
    if err != nil {
        t.Fatal(err)
    }

    // We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
    rr := httptest.NewRecorder()
    handler := http.HandlerFunc(HealthCheckHandler)

    // Our handlers satisfy http.Handler, so we can call their ServeHTTP method
    // directly and pass in our Request and ResponseRecorder.
    handler.ServeHTTP(rr, req)

    // Check the status code is what we expect.
    if status := rr.Code; status != http.StatusOK {
        t.Errorf("handler returned wrong status code: got %v want %v",
            status, http.StatusOK)
    }

    // Check the response body is what we expect.
    expected := `{"alive": true}`
    if rr.Body.String() != expected {
        t.Errorf("handler returned unexpected body: got %v want %v",
            rr.Body.String(), expected)
    }
}
```

In the case that our routes have [variables](#examples), we can pass those in the request. We could write
[table-driven tests](https://dave.cheney.net/2013/06/09/writing-table-driven-tests-in-go) to test multiple
possible route variables as needed.

```go
// endpoints.go
func main() {
    r := mux.NewRouter()
    // A route with a route variable:
    r.HandleFunc("/metrics/{type}", MetricsHandler)

    log.Fatal(http.ListenAndServe("localhost:8080", r))
}
```

Our test file, with a table-driven test of `routeVariables`:

```go
// endpoints_test.go
func TestMetricsHandler(t *testing.T) {
    tt := []struct{
        routeVariable string
        shouldPass bool
    }{
        {"goroutines", true},
        {"heap", true},
        {"counters", true},
        {"queries", true},
        {"adhadaeqm3k", false},
    }

    for _, tc := range tt {
        path := fmt.Sprintf("/metrics/%s", tc.routeVariable)
        req, err := http.NewRequest("GET", path, nil)
        if err != nil {
            t.Fatal(err)
        }

        rr := httptest.NewRecorder()
	
	// Need to create a router that we can pass the request through so that the vars will be added to the context
	router := mux.NewRouter()
        router.HandleFunc("/metrics/{type}", MetricsHandler)
        router.ServeHTTP(rr, req)

        // In this case, our MetricsHandler returns a non-200 response
        // for a route variable it doesn't know about.
        if rr.Code == http.StatusOK && !tc.shouldPass {
            t.Errorf("handler should have failed on routeVariable %s: got %v want %v",
                tc.routeVariable, rr.Code, http.StatusOK)
        }
    }
}
```

## Full Example

Here's a complete, runnable example of a small `mux` based server:

```go
package main

import (
    "net/http"
    "log"
    "github.com/gorilla/mux"
)

func YourHandler(w http.ResponseWriter, r *http.Request) {
    w.Write([]byte("Gorilla!\n"))
}

func main() {
    r := mux.NewRouter()
    // Routes consist of a path and a handler function.
    r.HandleFunc("/", YourHandler)

    // Bind to a port and pass our router in
    log.Fatal(http.ListenAndServe(":8000", r))
}
```

## License

BSD licensed. See the LICENSE file for details.
