// Copyright 2016 beego Author. All Rights Reserved.
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

package context

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestXsrfReset_01(t *testing.T) {
	r := &http.Request{}
	c := NewContext()
	c.Request = r
	c.ResponseWriter = &Response{}
	c.ResponseWriter.reset(httptest.NewRecorder())
	c.Output.Reset(c)
	c.Input.Reset(c)
	c.XSRFToken("key", 16)
	if c._xsrfToken == "" {
		t.FailNow()
	}
	token := c._xsrfToken
	c.Reset(&Response{ResponseWriter: httptest.NewRecorder()}, r)
	if c._xsrfToken != "" {
		t.FailNow()
	}
	c.XSRFToken("key", 16)
	if c._xsrfToken == "" {
		t.FailNow()
	}
	if token == c._xsrfToken {
		t.FailNow()
	}
}
