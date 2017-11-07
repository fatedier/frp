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

// Package captcha implements generation and verification of image CAPTCHAs.
// an example for use captcha
//
// ```
// package controllers
//
// import (
// 	"github.com/astaxie/beego"
// 	"github.com/astaxie/beego/cache"
// 	"github.com/astaxie/beego/utils/captcha"
// )
//
// var cpt *captcha.Captcha
//
// func init() {
// 	// use beego cache system store the captcha data
// 	store := cache.NewMemoryCache()
// 	cpt = captcha.NewWithFilter("/captcha/", store)
// }
//
// type MainController struct {
// 	beego.Controller
// }
//
// func (this *MainController) Get() {
// 	this.TplName = "index.tpl"
// }
//
// func (this *MainController) Post() {
// 	this.TplName = "index.tpl"
//
// 	this.Data["Success"] = cpt.VerifyReq(this.Ctx.Request)
// }
// ```
//
// template usage
//
// ```
// {{.Success}}
// <form action="/" method="post">
// 	{{create_captcha}}
// 	<input name="captcha" type="text">
// </form>
// ```
package captcha

import (
	"fmt"
	"html/template"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/cache"
	"github.com/astaxie/beego/context"
	"github.com/astaxie/beego/logs"
	"github.com/astaxie/beego/utils"
)

var (
	defaultChars = []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
)

const (
	// default captcha attributes
	challengeNums    = 6
	expiration       = 600 * time.Second
	fieldIDName      = "captcha_id"
	fieldCaptchaName = "captcha"
	cachePrefix      = "captcha_"
	defaultURLPrefix = "/captcha/"
)

// Captcha struct
type Captcha struct {
	// beego cache store
	store cache.Cache

	// url prefix for captcha image
	URLPrefix string

	// specify captcha id input field name
	FieldIDName string
	// specify captcha result input field name
	FieldCaptchaName string

	// captcha image width and height
	StdWidth  int
	StdHeight int

	// captcha chars nums
	ChallengeNums int

	// captcha expiration seconds
	Expiration time.Duration

	// cache key prefix
	CachePrefix string
}

// generate key string
func (c *Captcha) key(id string) string {
	return c.CachePrefix + id
}

// generate rand chars with default chars
func (c *Captcha) genRandChars() []byte {
	return utils.RandomCreateBytes(c.ChallengeNums, defaultChars...)
}

// Handler beego filter handler for serve captcha image
func (c *Captcha) Handler(ctx *context.Context) {
	var chars []byte

	id := path.Base(ctx.Request.RequestURI)
	if i := strings.Index(id, "."); i != -1 {
		id = id[:i]
	}

	key := c.key(id)

	if len(ctx.Input.Query("reload")) > 0 {
		chars = c.genRandChars()
		if err := c.store.Put(key, chars, c.Expiration); err != nil {
			ctx.Output.SetStatus(500)
			ctx.WriteString("captcha reload error")
			logs.Error("Reload Create Captcha Error:", err)
			return
		}
	} else {
		if v, ok := c.store.Get(key).([]byte); ok {
			chars = v
		} else {
			ctx.Output.SetStatus(404)
			ctx.WriteString("captcha not found")
			return
		}
	}

	img := NewImage(chars, c.StdWidth, c.StdHeight)
	if _, err := img.WriteTo(ctx.ResponseWriter); err != nil {
		logs.Error("Write Captcha Image Error:", err)
	}
}

// CreateCaptchaHTML template func for output html
func (c *Captcha) CreateCaptchaHTML() template.HTML {
	value, err := c.CreateCaptcha()
	if err != nil {
		logs.Error("Create Captcha Error:", err)
		return ""
	}

	// create html
	return template.HTML(fmt.Sprintf(`<input type="hidden" name="%s" value="%s">`+
		`<a class="captcha" href="javascript:">`+
		`<img onclick="this.src=('%s%s.png?reload='+(new Date()).getTime())" class="captcha-img" src="%s%s.png">`+
		`</a>`, c.FieldIDName, value, c.URLPrefix, value, c.URLPrefix, value))
}

// CreateCaptcha create a new captcha id
func (c *Captcha) CreateCaptcha() (string, error) {
	// generate captcha id
	id := string(utils.RandomCreateBytes(15))

	// get the captcha chars
	chars := c.genRandChars()

	// save to store
	if err := c.store.Put(c.key(id), chars, c.Expiration); err != nil {
		return "", err
	}

	return id, nil
}

// VerifyReq verify from a request
func (c *Captcha) VerifyReq(req *http.Request) bool {
	req.ParseForm()
	return c.Verify(req.Form.Get(c.FieldIDName), req.Form.Get(c.FieldCaptchaName))
}

// Verify direct verify id and challenge string
func (c *Captcha) Verify(id string, challenge string) (success bool) {
	if len(challenge) == 0 || len(id) == 0 {
		return
	}

	var chars []byte

	key := c.key(id)

	if v, ok := c.store.Get(key).([]byte); ok {
		chars = v
	} else {
		return
	}

	defer func() {
		// finally remove it
		c.store.Delete(key)
	}()

	if len(chars) != len(challenge) {
		return
	}
	// verify challenge
	for i, c := range chars {
		if c != challenge[i]-48 {
			return
		}
	}

	return true
}

// NewCaptcha create a new captcha.Captcha
func NewCaptcha(urlPrefix string, store cache.Cache) *Captcha {
	cpt := &Captcha{}
	cpt.store = store
	cpt.FieldIDName = fieldIDName
	cpt.FieldCaptchaName = fieldCaptchaName
	cpt.ChallengeNums = challengeNums
	cpt.Expiration = expiration
	cpt.CachePrefix = cachePrefix
	cpt.StdWidth = stdWidth
	cpt.StdHeight = stdHeight

	if len(urlPrefix) == 0 {
		urlPrefix = defaultURLPrefix
	}

	if urlPrefix[len(urlPrefix)-1] != '/' {
		urlPrefix += "/"
	}

	cpt.URLPrefix = urlPrefix

	return cpt
}

// NewWithFilter create a new captcha.Captcha and auto AddFilter for serve captacha image
// and add a template func for output html
func NewWithFilter(urlPrefix string, store cache.Cache) *Captcha {
	cpt := NewCaptcha(urlPrefix, store)

	// create filter for serve captcha image
	beego.InsertFilter(cpt.URLPrefix+"*", beego.BeforeRouter, cpt.Handler)

	// add to template func map
	beego.AddFuncMap("create_captcha", cpt.CreateCaptchaHTML)

	return cpt
}
