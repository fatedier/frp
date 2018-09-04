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

// Package xml for config provider.
//
// depend on github.com/beego/x2j.
//
// go install github.com/beego/x2j.
//
// Usage:
//  import(
//    _ "github.com/astaxie/beego/config/xml"
//      "github.com/astaxie/beego/config"
//  )
//
//  cnf, err := config.NewConfig("xml", "config.xml")
//
//More docs http://beego.me/docs/module/config.md
package xml

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/astaxie/beego/config"
	"github.com/beego/x2j"
)

// Config is a xml config parser and implements Config interface.
// xml configurations should be included in <config></config> tag.
// only support key/value pair as <key>value</key> as each item.
type Config struct{}

// Parse returns a ConfigContainer with parsed xml config map.
func (xc *Config) Parse(filename string) (config.Configer, error) {
	context, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	return xc.ParseData(context)
}

// ParseData xml data
func (xc *Config) ParseData(data []byte) (config.Configer, error) {
	x := &ConfigContainer{data: make(map[string]interface{})}

	d, err := x2j.DocToMap(string(data))
	if err != nil {
		return nil, err
	}

	x.data = config.ExpandValueEnvForMap(d["config"].(map[string]interface{}))

	return x, nil
}

// ConfigContainer A Config represents the xml configuration.
type ConfigContainer struct {
	data map[string]interface{}
	sync.Mutex
}

// Bool returns the boolean value for a given key.
func (c *ConfigContainer) Bool(key string) (bool, error) {
	if v := c.data[key]; v != nil {
		return config.ParseBool(v)
	}
	return false, fmt.Errorf("not exist key: %q", key)
}

// DefaultBool return the bool value if has no error
// otherwise return the defaultval
func (c *ConfigContainer) DefaultBool(key string, defaultval bool) bool {
	v, err := c.Bool(key)
	if err != nil {
		return defaultval
	}
	return v
}

// Int returns the integer value for a given key.
func (c *ConfigContainer) Int(key string) (int, error) {
	return strconv.Atoi(c.data[key].(string))
}

// DefaultInt returns the integer value for a given key.
// if err != nil return defaltval
func (c *ConfigContainer) DefaultInt(key string, defaultval int) int {
	v, err := c.Int(key)
	if err != nil {
		return defaultval
	}
	return v
}

// Int64 returns the int64 value for a given key.
func (c *ConfigContainer) Int64(key string) (int64, error) {
	return strconv.ParseInt(c.data[key].(string), 10, 64)
}

// DefaultInt64 returns the int64 value for a given key.
// if err != nil return defaltval
func (c *ConfigContainer) DefaultInt64(key string, defaultval int64) int64 {
	v, err := c.Int64(key)
	if err != nil {
		return defaultval
	}
	return v

}

// Float returns the float value for a given key.
func (c *ConfigContainer) Float(key string) (float64, error) {
	return strconv.ParseFloat(c.data[key].(string), 64)
}

// DefaultFloat returns the float64 value for a given key.
// if err != nil return defaltval
func (c *ConfigContainer) DefaultFloat(key string, defaultval float64) float64 {
	v, err := c.Float(key)
	if err != nil {
		return defaultval
	}
	return v
}

// String returns the string value for a given key.
func (c *ConfigContainer) String(key string) string {
	if v, ok := c.data[key].(string); ok {
		return v
	}
	return ""
}

// DefaultString returns the string value for a given key.
// if err != nil return defaltval
func (c *ConfigContainer) DefaultString(key string, defaultval string) string {
	v := c.String(key)
	if v == "" {
		return defaultval
	}
	return v
}

// Strings returns the []string value for a given key.
func (c *ConfigContainer) Strings(key string) []string {
	v := c.String(key)
	if v == "" {
		return nil
	}
	return strings.Split(v, ";")
}

// DefaultStrings returns the []string value for a given key.
// if err != nil return defaltval
func (c *ConfigContainer) DefaultStrings(key string, defaultval []string) []string {
	v := c.Strings(key)
	if v == nil {
		return defaultval
	}
	return v
}

// GetSection returns map for the given section
func (c *ConfigContainer) GetSection(section string) (map[string]string, error) {
	if v, ok := c.data[section].(map[string]interface{}); ok {
		mapstr := make(map[string]string)
		for k, val := range v {
			mapstr[k] = config.ToString(val)
		}
		return mapstr, nil
	}
	return nil, fmt.Errorf("section '%s' not found", section)
}

// SaveConfigFile save the config into file
func (c *ConfigContainer) SaveConfigFile(filename string) (err error) {
	// Write configuration file by filename.
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	b, err := xml.MarshalIndent(c.data, "  ", "    ")
	if err != nil {
		return err
	}
	_, err = f.Write(b)
	return err
}

// Set writes a new value for key.
func (c *ConfigContainer) Set(key, val string) error {
	c.Lock()
	defer c.Unlock()
	c.data[key] = val
	return nil
}

// DIY returns the raw value by a given key.
func (c *ConfigContainer) DIY(key string) (v interface{}, err error) {
	if v, ok := c.data[key]; ok {
		return v, nil
	}
	return nil, errors.New("not exist key")
}

func init() {
	config.Register("xml", &Config{})
}
