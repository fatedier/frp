// Copyright 2025 The frp Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package http

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/gorilla/mux"
)

type Context struct {
	Req  *http.Request
	Resp http.ResponseWriter
	vars map[string]string
}

func NewContext(w http.ResponseWriter, r *http.Request) *Context {
	return &Context{
		Req:  r,
		Resp: w,
		vars: mux.Vars(r),
	}
}

func (c *Context) Param(key string) string {
	return c.vars[key]
}

func (c *Context) Query(key string) string {
	return c.Req.URL.Query().Get(key)
}

func (c *Context) BindJSON(obj any) error {
	body, err := io.ReadAll(c.Req.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, obj)
}

func (c *Context) Body() ([]byte, error) {
	return io.ReadAll(c.Req.Body)
}
