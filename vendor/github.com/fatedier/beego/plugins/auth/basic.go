// Copyright 2014 beego Author. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package auth provides handlers to enable basic auth support.
// Simple Usage:
//	import(
//		"github.com/astaxie/beego"
//		"github.com/astaxie/beego/plugins/auth"
//	)
//
//	func main(){
//		// authenticate every request
//		beego.InsertFilter("*", beego.BeforeRouter,auth.Basic("username","secretpassword"))
//		beego.Run()
//	}
//
//
// Advanced Usage:
//
//	func SecretAuth(username, password string) bool {
//		return username == "astaxie" && password == "helloBeego"
//	}
//	authPlugin := auth.NewBasicAuthenticator(SecretAuth, "Authorization Required")
//	beego.InsertFilter("*", beego.BeforeRouter,authPlugin)
package auth

import (
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/context"
)

var defaultRealm = "Authorization Required"

// Basic is the http basic auth
func Basic(username string, password string) beego.FilterFunc {
	secrets := func(user, pass string) bool {
		return user == username && pass == password
	}
	return NewBasicAuthenticator(secrets, defaultRealm)
}

// NewBasicAuthenticator return the BasicAuth
func NewBasicAuthenticator(secrets SecretProvider, Realm string) beego.FilterFunc {
	return func(ctx *context.Context) {
		a := &BasicAuth{Secrets: secrets, Realm: Realm}
		if username := a.CheckAuth(ctx.Request); username == "" {
			a.RequireAuth(ctx.ResponseWriter, ctx.Request)
		}
	}
}

// SecretProvider is the SecretProvider function
type SecretProvider func(user, pass string) bool

// BasicAuth store the SecretProvider and Realm
type BasicAuth struct {
	Secrets SecretProvider
	Realm   string
}

// CheckAuth Checks the username/password combination from the request. Returns
// either an empty string (authentication failed) or the name of the
// authenticated user.
// Supports MD5 and SHA1 password entries
func (a *BasicAuth) CheckAuth(r *http.Request) string {
	s := strings.SplitN(r.Header.Get("Authorization"), " ", 2)
	if len(s) != 2 || s[0] != "Basic" {
		return ""
	}

	b, err := base64.StdEncoding.DecodeString(s[1])
	if err != nil {
		return ""
	}
	pair := strings.SplitN(string(b), ":", 2)
	if len(pair) != 2 {
		return ""
	}

	if a.Secrets(pair[0], pair[1]) {
		return pair[0]
	}
	return ""
}

// RequireAuth http.Handler for BasicAuth which initiates the authentication process
// (or requires reauthentication).
func (a *BasicAuth) RequireAuth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("WWW-Authenticate", `Basic realm="`+a.Realm+`"`)
	w.WriteHeader(401)
	w.Write([]byte("401 Unauthorized\n"))
}
