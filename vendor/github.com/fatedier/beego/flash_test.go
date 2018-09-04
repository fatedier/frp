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
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type TestFlashController struct {
	Controller
}

func (t *TestFlashController) TestWriteFlash() {
	flash := NewFlash()
	flash.Notice("TestFlashString")
	flash.Store(&t.Controller)
	// we choose to serve json because we don't want to load a template html file
	t.ServeJSON(true)
}

func TestFlashHeader(t *testing.T) {
	// create fake GET request
	r, _ := http.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	// setup the handler
	handler := NewControllerRegister()
	handler.Add("/", &TestFlashController{}, "get:TestWriteFlash")
	handler.ServeHTTP(w, r)

	// get the Set-Cookie value
	sc := w.Header().Get("Set-Cookie")
	// match for the expected header
	res := strings.Contains(sc, "BEEGO_FLASH=%00notice%23BEEGOFLASH%23TestFlashString%00")
	// validate the assertion
	if res != true {
		t.Errorf("TestFlashHeader() unable to validate flash message")
	}
}
