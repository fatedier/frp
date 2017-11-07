// Copyright 2014 beego Author. All Rights Reserved.
// Copyright 2017 Faissal Elamraoui. All Rights Reserved.
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
package env

import (
	"os"
	"testing"
)

func TestEnvGet(t *testing.T) {
	gopath := Get("GOPATH", "")
	if gopath != os.Getenv("GOPATH") {
		t.Error("expected GOPATH not empty.")
	}

	noExistVar := Get("NOEXISTVAR", "foo")
	if noExistVar != "foo" {
		t.Errorf("expected NOEXISTVAR to equal foo, got %s.", noExistVar)
	}
}

func TestEnvMustGet(t *testing.T) {
	gopath, err := MustGet("GOPATH")
	if err != nil {
		t.Error(err)
	}

	if gopath != os.Getenv("GOPATH") {
		t.Errorf("expected GOPATH to be the same, got %s.", gopath)
	}

	_, err = MustGet("NOEXISTVAR")
	if err == nil {
		t.Error("expected error to be non-nil")
	}
}

func TestEnvSet(t *testing.T) {
	Set("MYVAR", "foo")
	myVar := Get("MYVAR", "bar")
	if myVar != "foo" {
		t.Errorf("expected MYVAR to equal foo, got %s.", myVar)
	}
}

func TestEnvMustSet(t *testing.T) {
	err := MustSet("FOO", "bar")
	if err != nil {
		t.Error(err)
	}

	fooVar := os.Getenv("FOO")
	if fooVar != "bar" {
		t.Errorf("expected FOO variable to equal bar, got %s.", fooVar)
	}
}

func TestEnvGetAll(t *testing.T) {
	envMap := GetAll()
	if len(envMap) == 0 {
		t.Error("expected environment not empty.")
	}
}
