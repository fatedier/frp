// Copyright 2012 The Gorilla Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mux

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

// Route stores information to match a request and build URLs.
type Route struct {
	// Request handler for the route.
	handler http.Handler
	// If true, this route never matches: it is only used to build URLs.
	buildOnly bool
	// The name used to build URLs.
	name string
	// Error resulted from building a route.
	err error

	// "global" reference to all named routes
	namedRoutes map[string]*Route

	// config possibly passed in from `Router`
	routeConf
}

// SkipClean reports whether path cleaning is enabled for this route via
// Router.SkipClean.
func (r *Route) SkipClean() bool {
	return r.skipClean
}

// Match matches the route against the request.
func (r *Route) Match(req *http.Request, match *RouteMatch) bool {
	if r.buildOnly || r.err != nil {
		return false
	}

	var matchErr error

	// Match everything.
	for _, m := range r.matchers {
		if matched := m.Match(req, match); !matched {
			if _, ok := m.(methodMatcher); ok {
				matchErr = ErrMethodMismatch
				continue
			}

			// Ignore ErrNotFound errors. These errors arise from match call
			// to Subrouters.
			//
			// This prevents subsequent matching subrouters from failing to
			// run middleware. If not ignored, the middleware would see a
			// non-nil MatchErr and be skipped, even when there was a
			// matching route.
			if match.MatchErr == ErrNotFound {
				match.MatchErr = nil
			}

			matchErr = nil
			return false
		}
	}

	if matchErr != nil {
		match.MatchErr = matchErr
		return false
	}

	if match.MatchErr == ErrMethodMismatch {
		// We found a route which matches request method, clear MatchErr
		match.MatchErr = nil
		// Then override the mis-matched handler
		match.Handler = r.handler
	}

	// Yay, we have a match. Let's collect some info about it.
	if match.Route == nil {
		match.Route = r
	}
	if match.Handler == nil {
		match.Handler = r.handler
	}
	if match.Vars == nil {
		match.Vars = make(map[string]string)
	}

	// Set variables.
	r.regexp.setMatch(req, match, r)
	return true
}

// ----------------------------------------------------------------------------
// Route attributes
// ----------------------------------------------------------------------------

// GetError returns an error resulted from building the route, if any.
func (r *Route) GetError() error {
	return r.err
}

// BuildOnly sets the route to never match: it is only used to build URLs.
func (r *Route) BuildOnly() *Route {
	r.buildOnly = true
	return r
}

// Handler --------------------------------------------------------------------

// Handler sets a handler for the route.
func (r *Route) Handler(handler http.Handler) *Route {
	if r.err == nil {
		r.handler = handler
	}
	return r
}

// HandlerFunc sets a handler function for the route.
func (r *Route) HandlerFunc(f func(http.ResponseWriter, *http.Request)) *Route {
	return r.Handler(http.HandlerFunc(f))
}

// GetHandler returns the handler for the route, if any.
func (r *Route) GetHandler() http.Handler {
	return r.handler
}

// Name -----------------------------------------------------------------------

// Name sets the name for the route, used to build URLs.
// It is an error to call Name more than once on a route.
func (r *Route) Name(name string) *Route {
	if r.name != "" {
		r.err = fmt.Errorf("mux: route already has name %q, can't set %q",
			r.name, name)
	}
	if r.err == nil {
		r.name = name
		r.namedRoutes[name] = r
	}
	return r
}

// GetName returns the name for the route, if any.
func (r *Route) GetName() string {
	return r.name
}

// ----------------------------------------------------------------------------
// Matchers
// ----------------------------------------------------------------------------

// matcher types try to match a request.
type matcher interface {
	Match(*http.Request, *RouteMatch) bool
}

// addMatcher adds a matcher to the route.
func (r *Route) addMatcher(m matcher) *Route {
	if r.err == nil {
		r.matchers = append(r.matchers, m)
	}
	return r
}

// addRegexpMatcher adds a host or path matcher and builder to a route.
func (r *Route) addRegexpMatcher(tpl string, typ regexpType) error {
	if r.err != nil {
		return r.err
	}
	if typ == regexpTypePath || typ == regexpTypePrefix {
		if len(tpl) > 0 && tpl[0] != '/' {
			return fmt.Errorf("mux: path must start with a slash, got %q", tpl)
		}
		if r.regexp.path != nil {
			tpl = strings.TrimRight(r.regexp.path.template, "/") + tpl
		}
	}
	rr, err := newRouteRegexp(tpl, typ, routeRegexpOptions{
		strictSlash:    r.strictSlash,
		useEncodedPath: r.useEncodedPath,
	})
	if err != nil {
		return err
	}
	for _, q := range r.regexp.queries {
		if err = uniqueVars(rr.varsN, q.varsN); err != nil {
			return err
		}
	}
	if typ == regexpTypeHost {
		if r.regexp.path != nil {
			if err = uniqueVars(rr.varsN, r.regexp.path.varsN); err != nil {
				return err
			}
		}
		r.regexp.host = rr
	} else {
		if r.regexp.host != nil {
			if err = uniqueVars(rr.varsN, r.regexp.host.varsN); err != nil {
				return err
			}
		}
		if typ == regexpTypeQuery {
			r.regexp.queries = append(r.regexp.queries, rr)
		} else {
			r.regexp.path = rr
		}
	}
	r.addMatcher(rr)
	return nil
}

// Headers --------------------------------------------------------------------

// headerMatcher matches the request against header values.
type headerMatcher map[string]string

func (m headerMatcher) Match(r *http.Request, match *RouteMatch) bool {
	return matchMapWithString(m, r.Header, true)
}

// Headers adds a matcher for request header values.
// It accepts a sequence of key/value pairs to be matched. For example:
//
//     r := mux.NewRouter()
//     r.Headers("Content-Type", "application/json",
//               "X-Requested-With", "XMLHttpRequest")
//
// The above route will only match if both request header values match.
// If the value is an empty string, it will match any value if the key is set.
func (r *Route) Headers(pairs ...string) *Route {
	if r.err == nil {
		var headers map[string]string
		headers, r.err = mapFromPairsToString(pairs...)
		return r.addMatcher(headerMatcher(headers))
	}
	return r
}

// headerRegexMatcher matches the request against the route given a regex for the header
type headerRegexMatcher map[string]*regexp.Regexp

func (m headerRegexMatcher) Match(r *http.Request, match *RouteMatch) bool {
	return matchMapWithRegex(m, r.Header, true)
}

// HeadersRegexp accepts a sequence of key/value pairs, where the value has regex
// support. For example:
//
//     r := mux.NewRouter()
//     r.HeadersRegexp("Content-Type", "application/(text|json)",
//               "X-Requested-With", "XMLHttpRequest")
//
// The above route will only match if both the request header matches both regular expressions.
// If the value is an empty string, it will match any value if the key is set.
// Use the start and end of string anchors (^ and $) to match an exact value.
func (r *Route) HeadersRegexp(pairs ...string) *Route {
	if r.err == nil {
		var headers map[string]*regexp.Regexp
		headers, r.err = mapFromPairsToRegex(pairs...)
		return r.addMatcher(headerRegexMatcher(headers))
	}
	return r
}

// Host -----------------------------------------------------------------------

// Host adds a matcher for the URL host.
// It accepts a template with zero or more URL variables enclosed by {}.
// Variables can define an optional regexp pattern to be matched:
//
// - {name} matches anything until the next dot.
//
// - {name:pattern} matches the given regexp pattern.
//
// For example:
//
//     r := mux.NewRouter()
//     r.Host("www.example.com")
//     r.Host("{subdomain}.domain.com")
//     r.Host("{subdomain:[a-z]+}.domain.com")
//
// Variable names must be unique in a given route. They can be retrieved
// calling mux.Vars(request).
func (r *Route) Host(tpl string) *Route {
	r.err = r.addRegexpMatcher(tpl, regexpTypeHost)
	return r
}

// MatcherFunc ----------------------------------------------------------------

// MatcherFunc is the function signature used by custom matchers.
type MatcherFunc func(*http.Request, *RouteMatch) bool

// Match returns the match for a given request.
func (m MatcherFunc) Match(r *http.Request, match *RouteMatch) bool {
	return m(r, match)
}

// MatcherFunc adds a custom function to be used as request matcher.
func (r *Route) MatcherFunc(f MatcherFunc) *Route {
	return r.addMatcher(f)
}

// Methods --------------------------------------------------------------------

// methodMatcher matches the request against HTTP methods.
type methodMatcher []string

func (m methodMatcher) Match(r *http.Request, match *RouteMatch) bool {
	return matchInArray(m, r.Method)
}

// Methods adds a matcher for HTTP methods.
// It accepts a sequence of one or more methods to be matched, e.g.:
// "GET", "POST", "PUT".
func (r *Route) Methods(methods ...string) *Route {
	for k, v := range methods {
		methods[k] = strings.ToUpper(v)
	}
	return r.addMatcher(methodMatcher(methods))
}

// Path -----------------------------------------------------------------------

// Path adds a matcher for the URL path.
// It accepts a template with zero or more URL variables enclosed by {}. The
// template must start with a "/".
// Variables can define an optional regexp pattern to be matched:
//
// - {name} matches anything until the next slash.
//
// - {name:pattern} matches the given regexp pattern.
//
// For example:
//
//     r := mux.NewRouter()
//     r.Path("/products/").Handler(ProductsHandler)
//     r.Path("/products/{key}").Handler(ProductsHandler)
//     r.Path("/articles/{category}/{id:[0-9]+}").
//       Handler(ArticleHandler)
//
// Variable names must be unique in a given route. They can be retrieved
// calling mux.Vars(request).
func (r *Route) Path(tpl string) *Route {
	r.err = r.addRegexpMatcher(tpl, regexpTypePath)
	return r
}

// PathPrefix -----------------------------------------------------------------

// PathPrefix adds a matcher for the URL path prefix. This matches if the given
// template is a prefix of the full URL path. See Route.Path() for details on
// the tpl argument.
//
// Note that it does not treat slashes specially ("/foobar/" will be matched by
// the prefix "/foo") so you may want to use a trailing slash here.
//
// Also note that the setting of Router.StrictSlash() has no effect on routes
// with a PathPrefix matcher.
func (r *Route) PathPrefix(tpl string) *Route {
	r.err = r.addRegexpMatcher(tpl, regexpTypePrefix)
	return r
}

// Query ----------------------------------------------------------------------

// Queries adds a matcher for URL query values.
// It accepts a sequence of key/value pairs. Values may define variables.
// For example:
//
//     r := mux.NewRouter()
//     r.Queries("foo", "bar", "id", "{id:[0-9]+}")
//
// The above route will only match if the URL contains the defined queries
// values, e.g.: ?foo=bar&id=42.
//
// If the value is an empty string, it will match any value if the key is set.
//
// Variables can define an optional regexp pattern to be matched:
//
// - {name} matches anything until the next slash.
//
// - {name:pattern} matches the given regexp pattern.
func (r *Route) Queries(pairs ...string) *Route {
	length := len(pairs)
	if length%2 != 0 {
		r.err = fmt.Errorf(
			"mux: number of parameters must be multiple of 2, got %v", pairs)
		return nil
	}
	for i := 0; i < length; i += 2 {
		if r.err = r.addRegexpMatcher(pairs[i]+"="+pairs[i+1], regexpTypeQuery); r.err != nil {
			return r
		}
	}

	return r
}

// Schemes --------------------------------------------------------------------

// schemeMatcher matches the request against URL schemes.
type schemeMatcher []string

func (m schemeMatcher) Match(r *http.Request, match *RouteMatch) bool {
	return matchInArray(m, r.URL.Scheme)
}

// Schemes adds a matcher for URL schemes.
// It accepts a sequence of schemes to be matched, e.g.: "http", "https".
func (r *Route) Schemes(schemes ...string) *Route {
	for k, v := range schemes {
		schemes[k] = strings.ToLower(v)
	}
	if len(schemes) > 0 {
		r.buildScheme = schemes[0]
	}
	return r.addMatcher(schemeMatcher(schemes))
}

// BuildVarsFunc --------------------------------------------------------------

// BuildVarsFunc is the function signature used by custom build variable
// functions (which can modify route variables before a route's URL is built).
type BuildVarsFunc func(map[string]string) map[string]string

// BuildVarsFunc adds a custom function to be used to modify build variables
// before a route's URL is built.
func (r *Route) BuildVarsFunc(f BuildVarsFunc) *Route {
	if r.buildVarsFunc != nil {
		// compose the old and new functions
		old := r.buildVarsFunc
		r.buildVarsFunc = func(m map[string]string) map[string]string {
			return f(old(m))
		}
	} else {
		r.buildVarsFunc = f
	}
	return r
}

// Subrouter ------------------------------------------------------------------

// Subrouter creates a subrouter for the route.
//
// It will test the inner routes only if the parent route matched. For example:
//
//     r := mux.NewRouter()
//     s := r.Host("www.example.com").Subrouter()
//     s.HandleFunc("/products/", ProductsHandler)
//     s.HandleFunc("/products/{key}", ProductHandler)
//     s.HandleFunc("/articles/{category}/{id:[0-9]+}"), ArticleHandler)
//
// Here, the routes registered in the subrouter won't be tested if the host
// doesn't match.
func (r *Route) Subrouter() *Router {
	// initialize a subrouter with a copy of the parent route's configuration
	router := &Router{routeConf: copyRouteConf(r.routeConf), namedRoutes: r.namedRoutes}
	r.addMatcher(router)
	return router
}

// ----------------------------------------------------------------------------
// URL building
// ----------------------------------------------------------------------------

// URL builds a URL for the route.
//
// It accepts a sequence of key/value pairs for the route variables. For
// example, given this route:
//
//     r := mux.NewRouter()
//     r.HandleFunc("/articles/{category}/{id:[0-9]+}", ArticleHandler).
//       Name("article")
//
// ...a URL for it can be built using:
//
//     url, err := r.Get("article").URL("category", "technology", "id", "42")
//
// ...which will return an url.URL with the following path:
//
//     "/articles/technology/42"
//
// This also works for host variables:
//
//     r := mux.NewRouter()
//     r.Host("{subdomain}.domain.com").
//       HandleFunc("/articles/{category}/{id:[0-9]+}", ArticleHandler).
//       Name("article")
//
//     // url.String() will be "http://news.domain.com/articles/technology/42"
//     url, err := r.Get("article").URL("subdomain", "news",
//                                      "category", "technology",
//                                      "id", "42")
//
// All variables defined in the route are required, and their values must
// conform to the corresponding patterns.
func (r *Route) URL(pairs ...string) (*url.URL, error) {
	if r.err != nil {
		return nil, r.err
	}
	values, err := r.prepareVars(pairs...)
	if err != nil {
		return nil, err
	}
	var scheme, host, path string
	queries := make([]string, 0, len(r.regexp.queries))
	if r.regexp.host != nil {
		if host, err = r.regexp.host.url(values); err != nil {
			return nil, err
		}
		scheme = "http"
		if r.buildScheme != "" {
			scheme = r.buildScheme
		}
	}
	if r.regexp.path != nil {
		if path, err = r.regexp.path.url(values); err != nil {
			return nil, err
		}
	}
	for _, q := range r.regexp.queries {
		var query string
		if query, err = q.url(values); err != nil {
			return nil, err
		}
		queries = append(queries, query)
	}
	return &url.URL{
		Scheme:   scheme,
		Host:     host,
		Path:     path,
		RawQuery: strings.Join(queries, "&"),
	}, nil
}

// URLHost builds the host part of the URL for a route. See Route.URL().
//
// The route must have a host defined.
func (r *Route) URLHost(pairs ...string) (*url.URL, error) {
	if r.err != nil {
		return nil, r.err
	}
	if r.regexp.host == nil {
		return nil, errors.New("mux: route doesn't have a host")
	}
	values, err := r.prepareVars(pairs...)
	if err != nil {
		return nil, err
	}
	host, err := r.regexp.host.url(values)
	if err != nil {
		return nil, err
	}
	u := &url.URL{
		Scheme: "http",
		Host:   host,
	}
	if r.buildScheme != "" {
		u.Scheme = r.buildScheme
	}
	return u, nil
}

// URLPath builds the path part of the URL for a route. See Route.URL().
//
// The route must have a path defined.
func (r *Route) URLPath(pairs ...string) (*url.URL, error) {
	if r.err != nil {
		return nil, r.err
	}
	if r.regexp.path == nil {
		return nil, errors.New("mux: route doesn't have a path")
	}
	values, err := r.prepareVars(pairs...)
	if err != nil {
		return nil, err
	}
	path, err := r.regexp.path.url(values)
	if err != nil {
		return nil, err
	}
	return &url.URL{
		Path: path,
	}, nil
}

// GetPathTemplate returns the template used to build the
// route match.
// This is useful for building simple REST API documentation and for instrumentation
// against third-party services.
// An error will be returned if the route does not define a path.
func (r *Route) GetPathTemplate() (string, error) {
	if r.err != nil {
		return "", r.err
	}
	if r.regexp.path == nil {
		return "", errors.New("mux: route doesn't have a path")
	}
	return r.regexp.path.template, nil
}

// GetPathRegexp returns the expanded regular expression used to match route path.
// This is useful for building simple REST API documentation and for instrumentation
// against third-party services.
// An error will be returned if the route does not define a path.
func (r *Route) GetPathRegexp() (string, error) {
	if r.err != nil {
		return "", r.err
	}
	if r.regexp.path == nil {
		return "", errors.New("mux: route does not have a path")
	}
	return r.regexp.path.regexp.String(), nil
}

// GetQueriesRegexp returns the expanded regular expressions used to match the
// route queries.
// This is useful for building simple REST API documentation and for instrumentation
// against third-party services.
// An error will be returned if the route does not have queries.
func (r *Route) GetQueriesRegexp() ([]string, error) {
	if r.err != nil {
		return nil, r.err
	}
	if r.regexp.queries == nil {
		return nil, errors.New("mux: route doesn't have queries")
	}
	var queries []string
	for _, query := range r.regexp.queries {
		queries = append(queries, query.regexp.String())
	}
	return queries, nil
}

// GetQueriesTemplates returns the templates used to build the
// query matching.
// This is useful for building simple REST API documentation and for instrumentation
// against third-party services.
// An error will be returned if the route does not define queries.
func (r *Route) GetQueriesTemplates() ([]string, error) {
	if r.err != nil {
		return nil, r.err
	}
	if r.regexp.queries == nil {
		return nil, errors.New("mux: route doesn't have queries")
	}
	var queries []string
	for _, query := range r.regexp.queries {
		queries = append(queries, query.template)
	}
	return queries, nil
}

// GetMethods returns the methods the route matches against
// This is useful for building simple REST API documentation and for instrumentation
// against third-party services.
// An error will be returned if route does not have methods.
func (r *Route) GetMethods() ([]string, error) {
	if r.err != nil {
		return nil, r.err
	}
	for _, m := range r.matchers {
		if methods, ok := m.(methodMatcher); ok {
			return []string(methods), nil
		}
	}
	return nil, errors.New("mux: route doesn't have methods")
}

// GetHostTemplate returns the template used to build the
// route match.
// This is useful for building simple REST API documentation and for instrumentation
// against third-party services.
// An error will be returned if the route does not define a host.
func (r *Route) GetHostTemplate() (string, error) {
	if r.err != nil {
		return "", r.err
	}
	if r.regexp.host == nil {
		return "", errors.New("mux: route doesn't have a host")
	}
	return r.regexp.host.template, nil
}

// prepareVars converts the route variable pairs into a map. If the route has a
// BuildVarsFunc, it is invoked.
func (r *Route) prepareVars(pairs ...string) (map[string]string, error) {
	m, err := mapFromPairsToString(pairs...)
	if err != nil {
		return nil, err
	}
	return r.buildVars(m), nil
}

func (r *Route) buildVars(m map[string]string) map[string]string {
	if r.buildVarsFunc != nil {
		m = r.buildVarsFunc(m)
	}
	return m
}
