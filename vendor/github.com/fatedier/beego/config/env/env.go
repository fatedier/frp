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
	"fmt"
	"os"
	"strings"

	"github.com/astaxie/beego/utils"
)

var env *utils.BeeMap

func init() {
	env = utils.NewBeeMap()
	for _, e := range os.Environ() {
		splits := strings.Split(e, "=")
		env.Set(splits[0], os.Getenv(splits[0]))
	}
}

// Get returns a value by key.
// If the key does not exist, the default value will be returned.
func Get(key string, defVal string) string {
	if val := env.Get(key); val != nil {
		return val.(string)
	}
	return defVal
}

// MustGet returns a value by key.
// If the key does not exist, it will return an error.
func MustGet(key string) (string, error) {
	if val := env.Get(key); val != nil {
		return val.(string), nil
	}
	return "", fmt.Errorf("no env variable with %s", key)
}

// Set sets a value in the ENV copy.
// This does not affect the child process environment.
func Set(key string, value string) {
	env.Set(key, value)
}

// MustSet sets a value in the ENV copy and the child process environment.
// It returns an error in case the set operation failed.
func MustSet(key string, value string) error {
	err := os.Setenv(key, value)
	if err != nil {
		return err
	}
	env.Set(key, value)
	return nil
}

// GetAll returns all keys/values in the current child process environment.
func GetAll() map[string]string {
	items := env.Items()
	envs := make(map[string]string, env.Count())

	for key, val := range items {
		switch key := key.(type) {
		case string:
			switch val := val.(type) {
			case string:
				envs[key] = val
			}
		}
	}
	return envs
}
