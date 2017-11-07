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

package cache

import (
	"testing"
)

func TestGetString(t *testing.T) {
	var t1 = "test1"
	if "test1" != GetString(t1) {
		t.Error("get string from string error")
	}
	var t2 = []byte("test2")
	if "test2" != GetString(t2) {
		t.Error("get string from byte array error")
	}
	var t3 = 1
	if "1" != GetString(t3) {
		t.Error("get string from int error")
	}
	var t4 int64 = 1
	if "1" != GetString(t4) {
		t.Error("get string from int64 error")
	}
	var t5 = 1.1
	if "1.1" != GetString(t5) {
		t.Error("get string from float64 error")
	}

	if "" != GetString(nil) {
		t.Error("get string from nil error")
	}
}

func TestGetInt(t *testing.T) {
	var t1 = 1
	if 1 != GetInt(t1) {
		t.Error("get int from int error")
	}
	var t2 int32 = 32
	if 32 != GetInt(t2) {
		t.Error("get int from int32 error")
	}
	var t3 int64 = 64
	if 64 != GetInt(t3) {
		t.Error("get int from int64 error")
	}
	var t4 = "128"
	if 128 != GetInt(t4) {
		t.Error("get int from num string error")
	}
	if 0 != GetInt(nil) {
		t.Error("get int from nil error")
	}
}

func TestGetInt64(t *testing.T) {
	var i int64 = 1
	var t1 = 1
	if i != GetInt64(t1) {
		t.Error("get int64 from int error")
	}
	var t2 int32 = 1
	if i != GetInt64(t2) {
		t.Error("get int64 from int32 error")
	}
	var t3 int64 = 1
	if i != GetInt64(t3) {
		t.Error("get int64 from int64 error")
	}
	var t4 = "1"
	if i != GetInt64(t4) {
		t.Error("get int64 from num string error")
	}
	if 0 != GetInt64(nil) {
		t.Error("get int64 from nil")
	}
}

func TestGetFloat64(t *testing.T) {
	var f = 1.11
	var t1 float32 = 1.11
	if f != GetFloat64(t1) {
		t.Error("get float64 from float32 error")
	}
	var t2 = 1.11
	if f != GetFloat64(t2) {
		t.Error("get float64 from float64 error")
	}
	var t3 = "1.11"
	if f != GetFloat64(t3) {
		t.Error("get float64 from string error")
	}

	var f2 float64 = 1
	var t4 = 1
	if f2 != GetFloat64(t4) {
		t.Error("get float64 from int error")
	}

	if 0 != GetFloat64(nil) {
		t.Error("get float64 from nil error")
	}
}

func TestGetBool(t *testing.T) {
	var t1 = true
	if true != GetBool(t1) {
		t.Error("get bool from bool error")
	}
	var t2 = "true"
	if true != GetBool(t2) {
		t.Error("get bool from string error")
	}
	if false != GetBool(nil) {
		t.Error("get bool from nil error")
	}
}

func byteArrayEquals(a []byte, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}
