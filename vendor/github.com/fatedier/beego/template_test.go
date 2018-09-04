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
	"os"
	"path/filepath"
	"testing"
)

var header = `{{define "header"}}
<h1>Hello, astaxie!</h1>
{{end}}`

var index = `<!DOCTYPE html>
<html>
  <head>
    <title>beego welcome template</title>
  </head>
  <body>
{{template "block"}}
{{template "header"}}
{{template "blocks/block.tpl"}}
  </body>
</html>
`

var block = `{{define "block"}}
<h1>Hello, blocks!</h1>
{{end}}`

func TestTemplate(t *testing.T) {
	dir := "_beeTmp"
	files := []string{
		"header.tpl",
		"index.tpl",
		"blocks/block.tpl",
	}
	if err := os.MkdirAll(dir, 0777); err != nil {
		t.Fatal(err)
	}
	for k, name := range files {
		os.MkdirAll(filepath.Dir(filepath.Join(dir, name)), 0777)
		if f, err := os.Create(filepath.Join(dir, name)); err != nil {
			t.Fatal(err)
		} else {
			if k == 0 {
				f.WriteString(header)
			} else if k == 1 {
				f.WriteString(index)
			} else if k == 2 {
				f.WriteString(block)
			}

			f.Close()
		}
	}
	if err := AddViewPath(dir); err != nil {
		t.Fatal(err)
	}
	beeTemplates := beeViewPathTemplates[dir]
	if len(beeTemplates) != 3 {
		t.Fatalf("should be 3 but got %v", len(beeTemplates))
	}
	if err := beeTemplates["index.tpl"].ExecuteTemplate(os.Stdout, "index.tpl", nil); err != nil {
		t.Fatal(err)
	}
	for _, name := range files {
		os.RemoveAll(filepath.Join(dir, name))
	}
	os.RemoveAll(dir)
}

var menu = `<div class="menu">
<ul>
<li>menu1</li>
<li>menu2</li>
<li>menu3</li>
</ul>
</div>
`
var user = `<!DOCTYPE html>
<html>
  <head>
    <title>beego welcome template</title>
  </head>
  <body>
{{template "../public/menu.tpl"}}
  </body>
</html>
`

func TestRelativeTemplate(t *testing.T) {
	dir := "_beeTmp"

	//Just add dir to known viewPaths
	if err := AddViewPath(dir); err != nil {
		t.Fatal(err)
	}

	files := []string{
		"easyui/public/menu.tpl",
		"easyui/rbac/user.tpl",
	}
	if err := os.MkdirAll(dir, 0777); err != nil {
		t.Fatal(err)
	}
	for k, name := range files {
		os.MkdirAll(filepath.Dir(filepath.Join(dir, name)), 0777)
		if f, err := os.Create(filepath.Join(dir, name)); err != nil {
			t.Fatal(err)
		} else {
			if k == 0 {
				f.WriteString(menu)
			} else if k == 1 {
				f.WriteString(user)
			}
			f.Close()
		}
	}
	if err := BuildTemplate(dir, files[1]); err != nil {
		t.Fatal(err)
	}
	beeTemplates := beeViewPathTemplates[dir]
	if err := beeTemplates["easyui/rbac/user.tpl"].ExecuteTemplate(os.Stdout, "easyui/rbac/user.tpl", nil); err != nil {
		t.Fatal(err)
	}
	for _, name := range files {
		os.RemoveAll(filepath.Join(dir, name))
	}
	os.RemoveAll(dir)
}
