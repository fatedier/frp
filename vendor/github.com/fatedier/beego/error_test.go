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

package beego

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

type errorTestController struct {
	Controller
}

const parseCodeError = "parse code error"

func (ec *errorTestController) Get() {
	errorCode, err := ec.GetInt("code")
	if err != nil {
		ec.Abort(parseCodeError)
	}
	if errorCode != 0 {
		ec.CustomAbort(errorCode, ec.GetString("code"))
	}
	ec.Abort("404")
}

func TestErrorCode_01(t *testing.T) {
	registerDefaultErrorHandler()
	for k := range ErrorMaps {
		r, _ := http.NewRequest("GET", "/error?code="+k, nil)
		w := httptest.NewRecorder()

		handler := NewControllerRegister()
		handler.Add("/error", &errorTestController{})
		handler.ServeHTTP(w, r)
		code, _ := strconv.Atoi(k)
		if w.Code != code {
			t.Fail()
		}
		if !strings.Contains(string(w.Body.Bytes()), http.StatusText(code)) {
			t.Fail()
		}
	}
}

func TestErrorCode_02(t *testing.T) {
	registerDefaultErrorHandler()
	r, _ := http.NewRequest("GET", "/error?code=0", nil)
	w := httptest.NewRecorder()

	handler := NewControllerRegister()
	handler.Add("/error", &errorTestController{})
	handler.ServeHTTP(w, r)
	if w.Code != 404 {
		t.Fail()
	}
}

func TestErrorCode_03(t *testing.T) {
	registerDefaultErrorHandler()
	r, _ := http.NewRequest("GET", "/error?code=panic", nil)
	w := httptest.NewRecorder()

	handler := NewControllerRegister()
	handler.Add("/error", &errorTestController{})
	handler.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fail()
	}
	if string(w.Body.Bytes()) != parseCodeError {
		t.Fail()
	}
}
