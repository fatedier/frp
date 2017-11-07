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

package utils

import (
	"path/filepath"
	"reflect"
	"testing"
)

var noExistedFile = "/tmp/not_existed_file"

func TestSelfPath(t *testing.T) {
	path := SelfPath()
	if path == "" {
		t.Error("path cannot be empty")
	}
	t.Logf("SelfPath: %s", path)
}

func TestSelfDir(t *testing.T) {
	dir := SelfDir()
	t.Logf("SelfDir: %s", dir)
}

func TestFileExists(t *testing.T) {
	if !FileExists("./file.go") {
		t.Errorf("./file.go should exists, but it didn't")
	}

	if FileExists(noExistedFile) {
		t.Errorf("Weird, how could this file exists: %s", noExistedFile)
	}
}

func TestSearchFile(t *testing.T) {
	path, err := SearchFile(filepath.Base(SelfPath()), SelfDir())
	if err != nil {
		t.Error(err)
	}
	t.Log(path)

	_, err = SearchFile(noExistedFile, ".")
	if err == nil {
		t.Errorf("err shouldnot be nil, got path: %s", SelfDir())
	}
}

func TestGrepFile(t *testing.T) {
	_, err := GrepFile("", noExistedFile)
	if err == nil {
		t.Error("expect file-not-existed error, but got nothing")
	}

	path := filepath.Join(".", "testdata", "grepe.test")
	lines, err := GrepFile(`^\s*[^#]+`, path)
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(lines, []string{"hello", "world"}) {
		t.Errorf("expect [hello world], but receive %v", lines)
	}
}
