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

package beego

import (
	"fmt"
	"net/url"
	"strings"
)

// FlashData is a tools to maintain data when using across request.
type FlashData struct {
	Data map[string]string
}

// NewFlash return a new empty FlashData struct.
func NewFlash() *FlashData {
	return &FlashData{
		Data: make(map[string]string),
	}
}

// Set message to flash
func (fd *FlashData) Set(key string, msg string, args ...interface{}) {
	if len(args) == 0 {
		fd.Data[key] = msg
	} else {
		fd.Data[key] = fmt.Sprintf(msg, args...)
	}
}

// Success writes success message to flash.
func (fd *FlashData) Success(msg string, args ...interface{}) {
	if len(args) == 0 {
		fd.Data["success"] = msg
	} else {
		fd.Data["success"] = fmt.Sprintf(msg, args...)
	}
}

// Notice writes notice message to flash.
func (fd *FlashData) Notice(msg string, args ...interface{}) {
	if len(args) == 0 {
		fd.Data["notice"] = msg
	} else {
		fd.Data["notice"] = fmt.Sprintf(msg, args...)
	}
}

// Warning writes warning message to flash.
func (fd *FlashData) Warning(msg string, args ...interface{}) {
	if len(args) == 0 {
		fd.Data["warning"] = msg
	} else {
		fd.Data["warning"] = fmt.Sprintf(msg, args...)
	}
}

// Error writes error message to flash.
func (fd *FlashData) Error(msg string, args ...interface{}) {
	if len(args) == 0 {
		fd.Data["error"] = msg
	} else {
		fd.Data["error"] = fmt.Sprintf(msg, args...)
	}
}

// Store does the saving operation of flash data.
// the data are encoded and saved in cookie.
func (fd *FlashData) Store(c *Controller) {
	c.Data["flash"] = fd.Data
	var flashValue string
	for key, value := range fd.Data {
		flashValue += "\x00" + key + "\x23" + BConfig.WebConfig.FlashSeparator + "\x23" + value + "\x00"
	}
	c.Ctx.SetCookie(BConfig.WebConfig.FlashName, url.QueryEscape(flashValue), 0, "/")
}

// ReadFromRequest parsed flash data from encoded values in cookie.
func ReadFromRequest(c *Controller) *FlashData {
	flash := NewFlash()
	if cookie, err := c.Ctx.Request.Cookie(BConfig.WebConfig.FlashName); err == nil {
		v, _ := url.QueryUnescape(cookie.Value)
		vals := strings.Split(v, "\x00")
		for _, v := range vals {
			if len(v) > 0 {
				kv := strings.Split(v, "\x23"+BConfig.WebConfig.FlashSeparator+"\x23")
				if len(kv) == 2 {
					flash.Data[kv[0]] = kv[1]
				}
			}
		}
		//read one time then delete it
		c.Ctx.SetCookie(BConfig.WebConfig.FlashName, "", -1, "/")
	}
	c.Data["flash"] = flash.Data
	return flash
}
