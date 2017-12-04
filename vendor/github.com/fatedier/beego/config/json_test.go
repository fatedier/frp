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
	"os"
	"testing"
)

func TestJsonStartsWithArray(t *testing.T) {

	const jsoncontextwitharray = `[
	{
		"url": "user",
		"serviceAPI": "http://www.test.com/user"
	},
	{
		"url": "employee",
		"serviceAPI": "http://www.test.com/employee"
	}
]`
	f, err := os.Create("testjsonWithArray.conf")
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.WriteString(jsoncontextwitharray)
	if err != nil {
		f.Close()
		t.Fatal(err)
	}
	f.Close()
	defer os.Remove("testjsonWithArray.conf")
	jsonconf, err := NewConfig("json", "testjsonWithArray.conf")
	if err != nil {
		t.Fatal(err)
	}
	rootArray, err := jsonconf.DIY("rootArray")
	if err != nil {
		t.Error("array does not exist as element")
	}
	rootArrayCasted := rootArray.([]interface{})
	if rootArrayCasted == nil {
		t.Error("array from root is nil")
	} else {
		elem := rootArrayCasted[0].(map[string]interface{})
		if elem["url"] != "user" || elem["serviceAPI"] != "http://www.test.com/user" {
			t.Error("array[0] values are not valid")
		}

		elem2 := rootArrayCasted[1].(map[string]interface{})
		if elem2["url"] != "employee" || elem2["serviceAPI"] != "http://www.test.com/employee" {
			t.Error("array[1] values are not valid")
		}
	}
}

func TestJson(t *testing.T) {

	var (
		jsoncontext = `{
"appname": "beeapi",
"testnames": "foo;bar",
"httpport": 8080,
"mysqlport": 3600,
"PI": 3.1415976, 
"runmode": "dev",
"autorender": false,
"copyrequestbody": true,
"session": "on",
"cookieon": "off",
"newreg": "OFF",
"needlogin": "ON",
"enableSession": "Y",
"enableCookie": "N",
"flag": 1,
"path1": "${GOPATH}",
"path2": "${GOPATH||/home/go}",
"database": {
        "host": "host",
        "port": "port",
        "database": "database",
        "username": "username",
        "password": "${GOPATH}",
		"conns":{
			"maxconnection":12,
			"autoconnect":true,
			"connectioninfo":"info",
			"root": "${GOPATH}"
		}
    }
}`
		keyValue = map[string]interface{}{
			"appname":                         "beeapi",
			"testnames":                       []string{"foo", "bar"},
			"httpport":                        8080,
			"mysqlport":                       int64(3600),
			"PI":                              3.1415976,
			"runmode":                         "dev",
			"autorender":                      false,
			"copyrequestbody":                 true,
			"session":                         true,
			"cookieon":                        false,
			"newreg":                          false,
			"needlogin":                       true,
			"enableSession":                   true,
			"enableCookie":                    false,
			"flag":                            true,
			"path1":                           os.Getenv("GOPATH"),
			"path2":                           os.Getenv("GOPATH"),
			"database::host":                  "host",
			"database::port":                  "port",
			"database::database":              "database",
			"database::password":              os.Getenv("GOPATH"),
			"database::conns::maxconnection":  12,
			"database::conns::autoconnect":    true,
			"database::conns::connectioninfo": "info",
			"database::conns::root":           os.Getenv("GOPATH"),
			"unknown":                         "",
		}
	)

	f, err := os.Create("testjson.conf")
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.WriteString(jsoncontext)
	if err != nil {
		f.Close()
		t.Fatal(err)
	}
	f.Close()
	defer os.Remove("testjson.conf")
	jsonconf, err := NewConfig("json", "testjson.conf")
	if err != nil {
		t.Fatal(err)
	}

	for k, v := range keyValue {
		var err error
		var value interface{}
		switch v.(type) {
		case int:
			value, err = jsonconf.Int(k)
		case int64:
			value, err = jsonconf.Int64(k)
		case float64:
			value, err = jsonconf.Float(k)
		case bool:
			value, err = jsonconf.Bool(k)
		case []string:
			value = jsonconf.Strings(k)
		case string:
			value = jsonconf.String(k)
		default:
			value, err = jsonconf.DIY(k)
		}
		if err != nil {
			t.Fatalf("get key %q value fatal,%v err %s", k, v, err)
		} else if fmt.Sprintf("%v", v) != fmt.Sprintf("%v", value) {
			t.Fatalf("get key %q value, want %v got %v .", k, v, value)
		}

	}
	if err = jsonconf.Set("name", "astaxie"); err != nil {
		t.Fatal(err)
	}
	if jsonconf.String("name") != "astaxie" {
		t.Fatal("get name error")
	}

	if db, err := jsonconf.DIY("database"); err != nil {
		t.Fatal(err)
	} else if m, ok := db.(map[string]interface{}); !ok {
		t.Log(db)
		t.Fatal("db not map[string]interface{}")
	} else {
		if m["host"].(string) != "host" {
			t.Fatal("get host err")
		}
	}

	if _, err := jsonconf.Int("unknown"); err == nil {
		t.Error("unknown keys should return an error when expecting an Int")
	}

	if _, err := jsonconf.Int64("unknown"); err == nil {
		t.Error("unknown keys should return an error when expecting an Int64")
	}

	if _, err := jsonconf.Float("unknown"); err == nil {
		t.Error("unknown keys should return an error when expecting a Float")
	}

	if _, err := jsonconf.DIY("unknown"); err == nil {
		t.Error("unknown keys should return an error when expecting an interface{}")
	}

	if val := jsonconf.String("unknown"); val != "" {
		t.Error("unknown keys should return an empty string when expecting a String")
	}

	if _, err := jsonconf.Bool("unknown"); err == nil {
		t.Error("unknown keys should return an error when expecting a Bool")
	}

	if !jsonconf.DefaultBool("unknow", true) {
		t.Error("unknown keys with default value wrong")
	}
}
