package mux

import (
	"net/http"
	"strings"
)

// MiddlewareFunc is a function which receives an http.Handler and returns another http.Handler.
// Typically, the returned handler is a closure which does something with the http.ResponseWriter and http.Request passed
// to it, and then calls the handler passed as parameter to the MiddlewareFunc.
type MiddlewareFunc func(http.Handler) http.Handler

// middleware interface is anything which implements a MiddlewareFunc named Middleware.
type middleware interface {
	Middleware(handler http.Handler) http.Handler
}

// Middleware allows MiddlewareFunc to implement the middleware interface.
func (mw MiddlewareFunc) Middleware(handler http.Handler) http.Handler {
	return mw(handler)
}

// Use appends a MiddlewareFunc to the chain. Middleware can be used to intercept or otherwise modify requests and/or responses, and are executed in the order that they are applied to the Router.
func (r *Router) Use(mwf ...MiddlewareFunc) {
	for _, fn := range mwf {
		r.middlewares = append(r.middlewares, fn)
	}
}

// useInterface appends a middleware to the chain. Middleware can be used to intercept or otherwise modify requests and/or responses, and are executed in the order that they are applied to the Router.
func (r *Router) useInterface(mw middleware) {
	r.middlewares = append(r.middlewares, mw)
}

// CORSMethodMiddleware automatically sets the Access-Control-Allow-Methods response header
// on requests for routes that have an OPTIONS method matcher to all the method matchers on
// the route. Routes that do not explicitly handle OPTIONS requests will not be processed
// by the middleware. See examples for usage.
func CORSMethodMiddleware(r *Router) MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			allMethods, err := getAllMethodsForRoute(r, req)
			if err == nil {
				for _, v := range allMethods {
					if v == http.MethodOptions {
						w.Header().Set("Access-Control-Allow-Methods", strings.Join(allMethods, ","))
					}
				}
			}

			next.ServeHTTP(w, req)
		})
	}
}

// getAllMethodsForRoute returns all the methods from method matchers matching a given
// request.
func getAllMethodsForRoute(r *Router, req *http.Request) ([]string, error) {
	var allMethods []string

	err := r.Walk(func(route *Route, _ *Router, _ []*Route) error {
		for _, m := range route.matchers {
			if _, ok := m.(*routeRegexp); ok {
				if m.Match(req, &RouteMatch{}) {
					methods, err := route.GetMethods()
					if err != nil {
						return err
					}

					allMethods = append(allMethods, methods...)
				}
				break
			}
		}
		return nil
	})

	return allMethods, err
}
