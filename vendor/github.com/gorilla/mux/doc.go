// Copyright 2012 The Gorilla Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
Package mux implements a request router and dispatcher.

The name mux stands for "HTTP request multiplexer". Like the standard
http.ServeMux, mux.Router matches incoming requests against a list of
registered routes and calls a handler for the route that matches the URL
or other conditions. The main features are:

	* Requests can be matched based on URL host, path, path prefix, schemes,
	  header and query values, HTTP methods or using custom matchers.
	* URL hosts, paths and query values can have variables with an optional
	  regular expression.
	* Registered URLs can be built, or "reversed", which helps maintaining
	  references to resources.
	* Routes can be used as subrouters: nested routes are only tested if the
	  parent route matches. This is useful to define groups of routes that
	  share common conditions like a host, a path prefix or other repeated
	  attributes. As a bonus, this optimizes request matching.
	* It implements the http.Handler interface so it is compatible with the
	  standard http.ServeMux.

Let's start registering a couple of URL paths and handlers:

	func main() {
		r := mux.NewRouter()
		r.HandleFunc("/", HomeHandler)
		r.HandleFunc("/products", ProductsHandler)
		r.HandleFunc("/articles", ArticlesHandler)
		http.Handle("/", r)
	}

Here we register three routes mapping URL paths to handlers. This is
equivalent to how http.HandleFunc() works: if an incoming request URL matches
one of the paths, the corresponding handler is called passing
(http.ResponseWriter, *http.Request) as parameters.

Paths can have variables. They are defined using the format {name} or
{name:pattern}. If a regular expression pattern is not defined, the matched
variable will be anything until the next slash. For example:

	r := mux.NewRouter()
	r.HandleFunc("/products/{key}", ProductHandler)
	r.HandleFunc("/articles/{category}/", ArticlesCategoryHandler)
	r.HandleFunc("/articles/{category}/{id:[0-9]+}", ArticleHandler)

Groups can be used inside patterns, as long as they are non-capturing (?:re). For example:

	r.HandleFunc("/articles/{category}/{sort:(?:asc|desc|new)}", ArticlesCategoryHandler)

The names are used to create a map of route variables which can be retrieved
calling mux.Vars():

	vars := mux.Vars(request)
	category := vars["category"]

Note that if any capturing groups are present, mux will panic() during parsing. To prevent
this, convert any capturing groups to non-capturing, e.g. change "/{sort:(asc|desc)}" to
"/{sort:(?:asc|desc)}". This is a change from prior versions which behaved unpredictably
when capturing groups were present.

And this is all you need to know about the basic usage. More advanced options
are explained below.

Routes can also be restricted to a domain or subdomain. Just define a host
pattern to be matched. They can also have variables:

	r := mux.NewRouter()
	// Only matches if domain is "www.example.com".
	r.Host("www.example.com")
	// Matches a dynamic subdomain.
	r.Host("{subdomain:[a-z]+}.domain.com")

There are several other matchers that can be added. To match path prefixes:

	r.PathPrefix("/products/")

...or HTTP methods:

	r.Methods("GET", "POST")

...or URL schemes:

	r.Schemes("https")

...or header values:

	r.Headers("X-Requested-With", "XMLHttpRequest")

...or query values:

	r.Queries("key", "value")

...or to use a custom matcher function:

	r.MatcherFunc(func(r *http.Request, rm *RouteMatch) bool {
		return r.ProtoMajor == 0
	})

...and finally, it is possible to combine several matchers in a single route:

	r.HandleFunc("/products", ProductsHandler).
	  Host("www.example.com").
	  Methods("GET").
	  Schemes("http")

Setting the same matching conditions again and again can be boring, so we have
a way to group several routes that share the same requirements.
We call it "subrouting".

For example, let's say we have several URLs that should only match when the
host is "www.example.com". Create a route for that host and get a "subrouter"
from it:

	r := mux.NewRouter()
	s := r.Host("www.example.com").Subrouter()

Then register routes in the subrouter:

	s.HandleFunc("/products/", ProductsHandler)
	s.HandleFunc("/products/{key}", ProductHandler)
	s.HandleFunc("/articles/{category}/{id:[0-9]+}"), ArticleHandler)

The three URL paths we registered above will only be tested if the domain is
"www.example.com", because the subrouter is tested first. This is not
only convenient, but also optimizes request matching. You can create
subrouters combining any attribute matchers accepted by a route.

Subrouters can be used to create domain or path "namespaces": you define
subrouters in a central place and then parts of the app can register its
paths relatively to a given subrouter.

There's one more thing about subroutes. When a subrouter has a path prefix,
the inner routes use it as base for their paths:

	r := mux.NewRouter()
	s := r.PathPrefix("/products").Subrouter()
	// "/products/"
	s.HandleFunc("/", ProductsHandler)
	// "/products/{key}/"
	s.HandleFunc("/{key}/", ProductHandler)
	// "/products/{key}/details"
	s.HandleFunc("/{key}/details", ProductDetailsHandler)

Note that the path provided to PathPrefix() represents a "wildcard": calling
PathPrefix("/static/").Handler(...) means that the handler will be passed any
request that matches "/static/*". This makes it easy to serve static files with mux:

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

Now let's see how to build registered URLs.

Routes can be named. All routes that define a name can have their URLs built,
or "reversed". We define a name calling Name() on a route. For example:

	r := mux.NewRouter()
	r.HandleFunc("/articles/{category}/{id:[0-9]+}", ArticleHandler).
	  Name("article")

To build a URL, get the route and call the URL() method, passing a sequence of
key/value pairs for the route variables. For the previous route, we would do:

	url, err := r.Get("article").URL("category", "technology", "id", "42")

...and the result will be a url.URL with the following path:

	"/articles/technology/42"

This also works for host and query value variables:

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

All variables defined in the route are required, and their values must
conform to the corresponding patterns. These requirements guarantee that a
generated URL will always match a registered route -- the only exception is
for explicitly defined "build-only" routes which never match.

Regex support also exists for matching Headers within a route. For example, we could do:

	r.HeadersRegexp("Content-Type", "application/(text|json)")

...and the route will match both requests with a Content-Type of `application/json` as well as
`application/text`

There's also a way to build only the URL host or path for a route:
use the methods URLHost() or URLPath() instead. For the previous route,
we would do:

	// "http://news.domain.com/"
	host, err := r.Get("article").URLHost("subdomain", "news")

	// "/articles/technology/42"
	path, err := r.Get("article").URLPath("category", "technology", "id", "42")

And if you use subrouters, host and path defined separately can be built
as well:

	r := mux.NewRouter()
	s := r.Host("{subdomain}.domain.com").Subrouter()
	s.Path("/articles/{category}/{id:[0-9]+}").
	  HandlerFunc(ArticleHandler).
	  Name("article")

	// "http://news.domain.com/articles/technology/42"
	url, err := r.Get("article").URL("subdomain", "news",
	                                 "category", "technology",
	                                 "id", "42")

Mux supports the addition of middlewares to a Router, which are executed in the order they are added if a match is found, including its subrouters. Middlewares are (typically) small pieces of code which take one request, do something with it, and pass it down to another middleware or the final handler. Some common use cases for middleware are request logging, header manipulation, or ResponseWriter hijacking.

	type MiddlewareFunc func(http.Handler) http.Handler

Typically, the returned handler is a closure which does something with the http.ResponseWriter and http.Request passed to it, and then calls the handler passed as parameter to the MiddlewareFunc (closures can access variables from the context where they are created).

A very basic middleware which logs the URI of the request being handled could be written as:

	func simpleMw(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Do stuff here
			log.Println(r.RequestURI)
			// Call the next handler, which can be another middleware in the chain, or the final handler.
			next.ServeHTTP(w, r)
		})
	}

Middlewares can be added to a router using `Router.Use()`:

	r := mux.NewRouter()
	r.HandleFunc("/", handler)
	r.Use(simpleMw)

A more complex authentication middleware, which maps session token to users, could be written as:

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
				next.ServeHTTP(w, r)
			} else {
				http.Error(w, "Forbidden", http.StatusForbidden)
			}
		})
	}

	r := mux.NewRouter()
	r.HandleFunc("/", handler)

	amw := authenticationMiddleware{tokenUsers: make(map[string]string)}
	amw.Populate()

	r.Use(amw.Middleware)

Note: The handler chain will be stopped if your middleware doesn't call `next.ServeHTTP()` with the corresponding parameters. This can be used to abort a request if the middleware writer wants to.

*/
package mux
