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

package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

func TestIni(t *testing.T) {

	var (
		inicontext = `
;comment one
#comment two
appname = beeapi
httpport = 8080
mysqlport = 3600
PI = 3.1415976
runmode = "dev"
autorender = false
copyrequestbody = true
session= on
cookieon= off
newreg = OFF
needlogin = ON
enableSession = Y
enableCookie = N
flag = 1
path1 = ${GOPATH}
path2 = ${GOPATH||/home/go}
[demo]
key1="asta"
key2 = "xie"
CaseInsensitive = true
peers = one;two;three
password = ${GOPATH}
`

		keyValue = map[string]interface{}{
			"appname":               "beeapi",
			"httpport":              8080,
			"mysqlport":             int64(3600),
			"pi":                    3.1415976,
			"runmode":               "dev",
			"autorender":            false,
			"copyrequestbody":       true,
			"session":               true,
			"cookieon":              false,
			"newreg":                false,
			"needlogin":             true,
			"enableSession":         true,
			"enableCookie":          false,
			"flag":                  true,
			"path1":                 os.Getenv("GOPATH"),
			"path2":                 os.Getenv("GOPATH"),
			"demo::key1":            "asta",
			"demo::key2":            "xie",
			"demo::CaseInsensitive": true,
			"demo::peers":           []string{"one", "two", "three"},
			"demo::password":        os.Getenv("GOPATH"),
			"null":                  "",
			"demo2::key1":           "",
			"error":                 "",
			"emptystrings":          []string{},
		}
	)

	f, err := os.Create("testini.conf")
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.WriteString(inicontext)
	if err != nil {
		f.Close()
		t.Fatal(err)
	}
	f.Close()
	defer os.Remove("testini.conf")
	iniconf, err := NewConfig("ini", "testini.conf")
	if err != nil {
		t.Fatal(err)
	}
	for k, v := range keyValue {
		var err error
		var value interface{}
		switch v.(type) {
		case int:
			value, err = iniconf.Int(k)
		case int64:
			value, err = iniconf.Int64(k)
		case float64:
			value, err = iniconf.Float(k)
		case bool:
			value, err = iniconf.Bool(k)
		case []string:
			value = iniconf.Strings(k)
		case string:
			value = iniconf.String(k)
		default:
			value, err = iniconf.DIY(k)
		}
		if err != nil {
			t.Fatalf("get key %q value fail,err %s", k, err)
		} else if fmt.Sprintf("%v", v) != fmt.Sprintf("%v", value) {
			t.Fatalf("get key %q value, want %v got %v .", k, v, value)
		}

	}
	if err = iniconf.Set("name", "astaxie"); err != nil {
		t.Fatal(err)
	}
	if iniconf.String("name") != "astaxie" {
		t.Fatal("get name error")
	}

}

func TestIniSave(t *testing.T) {

	const (
		inicontext = `
app = app
;comment one
#comment two
# comment three
appname = beeapi
httpport = 8080
# DB Info
# enable db
[dbinfo]
# db type name
# suport mysql,sqlserver
name = mysql
`

		saveResult = `
app=app
#comment one
#comment two
# comment three
appname=beeapi
httpport=8080

# DB Info
# enable db
[dbinfo]
# db type name
# suport mysql,sqlserver
name=mysql
`
	)
	cfg, err := NewConfigData("ini", []byte(inicontext))
	if err != nil {
		t.Fatal(err)
	}
	name := "newIniConfig.ini"
	if err := cfg.SaveConfigFile(name); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(name)

	if data, err := ioutil.ReadFile(name); err != nil {
		t.Fatal(err)
	} else {
		cfgData := string(data)
		datas := strings.Split(saveResult, "\n")
		for _, line := range datas {
			if strings.Contains(cfgData, line+"\n") == false {
				t.Fatalf("different after save ini config file. need contains %q", line)
			}
		}

	}
}
