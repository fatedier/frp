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

// Package apiauth provides handlers to enable apiauth support.
//
// Simple Usage:
//	import(
//		"github.com/astaxie/beego"
//		"github.com/astaxie/beego/plugins/apiauth"
//	)
//
//	func main(){
//		// apiauth every request
//		beego.InsertFilter("*", beego.BeforeRouter,apiauth.APIBaiscAuth("appid","appkey"))
//		beego.Run()
//	}
//
// Advanced Usage:
//
//	func getAppSecret(appid string) string {
//		// get appsecret by appid
//		// maybe store in configure, maybe in database
//	}
//
//	beego.InsertFilter("*", beego.BeforeRouter,apiauth.APISecretAuth(getAppSecret, 360))
//
// Information:
//
// In the request user should include these params in the query
//
// 1. appid
//
//		 appid is assigned to the application
//
// 2. signature
//
//	get the signature use apiauth.Signature()
//
//	when you send to server remember use url.QueryEscape()
//
// 3. timestamp:
//
//       send the request time, the format is yyyy-mm-dd HH:ii:ss
//
package apiauth

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/url"
	"sort"
	"time"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/context"
)

// AppIDToAppSecret is used to get appsecret throw appid
type AppIDToAppSecret func(string) string

// APIBaiscAuth use the basic appid/appkey as the AppIdToAppSecret
func APIBaiscAuth(appid, appkey string) beego.FilterFunc {
	ft := func(aid string) string {
		if aid == appid {
			return appkey
		}
		return ""
	}
	return APISecretAuth(ft, 300)
}

// APISecretAuth use AppIdToAppSecret verify and
func APISecretAuth(f AppIDToAppSecret, timeout int) beego.FilterFunc {
	return func(ctx *context.Context) {
		if ctx.Input.Query("appid") == "" {
			ctx.ResponseWriter.WriteHeader(403)
			ctx.WriteString("miss query param: appid")
			return
		}
		appsecret := f(ctx.Input.Query("appid"))
		if appsecret == "" {
			ctx.ResponseWriter.WriteHeader(403)
			ctx.WriteString("not exist this appid")
			return
		}
		if ctx.Input.Query("signature") == "" {
			ctx.ResponseWriter.WriteHeader(403)
			ctx.WriteString("miss query param: signature")
			return
		}
		if ctx.Input.Query("timestamp") == "" {
			ctx.ResponseWriter.WriteHeader(403)
			ctx.WriteString("miss query param: timestamp")
			return
		}
		u, err := time.Parse("2006-01-02 15:04:05", ctx.Input.Query("timestamp"))
		if err != nil {
			ctx.ResponseWriter.WriteHeader(403)
			ctx.WriteString("timestamp format is error, should 2006-01-02 15:04:05")
			return
		}
		t := time.Now()
		if t.Sub(u).Seconds() > float64(timeout) {
			ctx.ResponseWriter.WriteHeader(403)
			ctx.WriteString("timeout! the request time is long ago, please try again")
			return
		}
		if ctx.Input.Query("signature") !=
			Signature(appsecret, ctx.Input.Method(), ctx.Request.Form, ctx.Input.URL()) {
			ctx.ResponseWriter.WriteHeader(403)
			ctx.WriteString("auth failed")
		}
	}
}

// Signature used to generate signature with the appsecret/method/params/RequestURI
func Signature(appsecret, method string, params url.Values, RequestURL string) (result string) {
	var b bytes.Buffer
	keys := make([]string, len(params))
	pa := make(map[string]string)
	for k, v := range params {
		pa[k] = v[0]
		keys = append(keys, k)
	}

	sort.Strings(keys)

	for _, key := range keys {
		if key == "signature" {
			continue
		}

		val := pa[key]
		if key != "" && val != "" {
			b.WriteString(key)
			b.WriteString(val)
		}
	}

	stringToSign := fmt.Sprintf("%v\n%v\n%v\n", method, b.String(), RequestURL)

	sha256 := sha256.New
	hash := hmac.New(sha256, []byte(appsecret))
	hash.Write([]byte(stringToSign))
	return base64.StdEncoding.EncodeToString(hash.Sum(nil))
}
