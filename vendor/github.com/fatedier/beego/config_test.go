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
	"encoding/json"
	"reflect"
	"testing"

	"github.com/astaxie/beego/config"
)

func TestDefaults(t *testing.T) {
	if BConfig.WebConfig.FlashName != "BEEGO_FLASH" {
		t.Errorf("FlashName was not set to default.")
	}

	if BConfig.WebConfig.FlashSeparator != "BEEGOFLASH" {
		t.Errorf("FlashName was not set to default.")
	}
}

func TestAssignConfig_01(t *testing.T) {
	_BConfig := &Config{}
	_BConfig.AppName = "beego_test"
	jcf := &config.JSONConfig{}
	ac, _ := jcf.ParseData([]byte(`{"AppName":"beego_json"}`))
	assignSingleConfig(_BConfig, ac)
	if _BConfig.AppName != "beego_json" {
		t.Log(_BConfig)
		t.FailNow()
	}
}

func TestAssignConfig_02(t *testing.T) {
	_BConfig := &Config{}
	bs, _ := json.Marshal(newBConfig())

	jsonMap := map[string]interface{}{}
	json.Unmarshal(bs, &jsonMap)

	configMap := map[string]interface{}{}
	for k, v := range jsonMap {
		if reflect.TypeOf(v).Kind() == reflect.Map {
			for k1, v1 := range v.(map[string]interface{}) {
				if reflect.TypeOf(v1).Kind() == reflect.Map {
					for k2, v2 := range v1.(map[string]interface{}) {
						configMap[k2] = v2
					}
				} else {
					configMap[k1] = v1
				}
			}
		} else {
			configMap[k] = v
		}
	}
	configMap["MaxMemory"] = 1024
	configMap["Graceful"] = true
	configMap["XSRFExpire"] = 32
	configMap["SessionProviderConfig"] = "file"
	configMap["FileLineNum"] = true

	jcf := &config.JSONConfig{}
	bs, _ = json.Marshal(configMap)
	ac, _ := jcf.ParseData([]byte(bs))

	for _, i := range []interface{}{_BConfig, &_BConfig.Listen, &_BConfig.WebConfig, &_BConfig.Log, &_BConfig.WebConfig.Session} {
		assignSingleConfig(i, ac)
	}

	if _BConfig.MaxMemory != 1024 {
		t.Log(_BConfig.MaxMemory)
		t.FailNow()
	}

	if !_BConfig.Listen.Graceful {
		t.Log(_BConfig.Listen.Graceful)
		t.FailNow()
	}

	if _BConfig.WebConfig.XSRFExpire != 32 {
		t.Log(_BConfig.WebConfig.XSRFExpire)
		t.FailNow()
	}

	if _BConfig.WebConfig.Session.SessionProviderConfig != "file" {
		t.Log(_BConfig.WebConfig.Session.SessionProviderConfig)
		t.FailNow()
	}

	if !_BConfig.Log.FileLineNum {
		t.Log(_BConfig.Log.FileLineNum)
		t.FailNow()
	}

}

func TestAssignConfig_03(t *testing.T) {
	jcf := &config.JSONConfig{}
	ac, _ := jcf.ParseData([]byte(`{"AppName":"beego"}`))
	ac.Set("AppName", "test_app")
	ac.Set("RunMode", "online")
	ac.Set("StaticDir", "download:down download2:down2")
	ac.Set("StaticExtensionsToGzip", ".css,.js,.html,.jpg,.png")
	assignConfig(ac)

	t.Logf("%#v", BConfig)

	if BConfig.AppName != "test_app" {
		t.FailNow()
	}

	if BConfig.RunMode != "online" {
		t.FailNow()
	}
	if BConfig.WebConfig.StaticDir["/download"] != "down" {
		t.FailNow()
	}
	if BConfig.WebConfig.StaticDir["/download2"] != "down2" {
		t.FailNow()
	}
	if len(BConfig.WebConfig.StaticExtensionsToGzip) != 5 {
		t.FailNow()
	}
}
