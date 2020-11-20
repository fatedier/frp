// Copyright 2016 fatedier, fatedier@gmail.com
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"text/template"
)

func GetMapWithoutPrefix(set map[string]string, prefix string) map[string]string {
	m := make(map[string]string)

	for key, value := range set {
		if strings.HasPrefix(key, prefix) {
			m[strings.TrimPrefix(key, prefix)] = value
		}
	}

	if len(m) == 0 {
		return nil
	}

	return m
}

func GetMapByPrefix(set map[string]string, prefix string) map[string]string {
	m := make(map[string]string)

	for key, value := range set {
		if strings.HasPrefix(key, prefix) {
			m[key] = value
		}
	}

	if len(m) == 0 {
		return nil
	}

	return m
}

// Render Env Values
var glbEnvs map[string]string

func init() {
	glbEnvs = make(map[string]string)
	envs := os.Environ()
	for _, env := range envs {
		kv := strings.Split(env, "=")
		if len(kv) != 2 {
			continue
		}
		glbEnvs[kv[0]] = kv[1]
	}
}

type Values struct {
	Envs map[string]string // environment vars
}

func GetValues() *Values {
	return &Values{
		Envs: glbEnvs,
	}
}

func RenderContent(in []byte) ([]byte, error) {
	tmpl, err := template.New("frp").Parse(string(in))
	if err != nil {
		return nil, err
	}

	buffer := bytes.NewBufferString("")
	v := GetValues()
	err = tmpl.Execute(buffer, v)
	if err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func GetRenderedConfFromFile(path string) ([]byte, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return RenderContent(data)
}

// BandwidthQuantity
const (
	MB = 1024 * 1024
	KB = 1024
)

type BandwidthQuantity struct {
	s string // MB or KB

	i int64 // bytes
}

func NewBandwidthQuantity(s string) (BandwidthQuantity, error) {
	q := BandwidthQuantity{}
	err := q.UnmarshalString(s)
	if err != nil {
		return q, err
	}
	return q, nil
}

func MustBandwidthQuantity(s string) BandwidthQuantity {
	q := BandwidthQuantity{}
	err := q.UnmarshalString(s)
	if err != nil {
		panic(err)
	}
	return q
}

func (q *BandwidthQuantity) Equal(u *BandwidthQuantity) bool {
	if q == nil && u == nil {
		return true
	}
	if q != nil && u != nil {
		return q.i == u.i
	}
	return false
}

func (q *BandwidthQuantity) String() string {
	return q.s
}

func (q *BandwidthQuantity) UnmarshalString(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}

	var (
		base int64
		f    float64
		err  error
	)
	if strings.HasSuffix(s, "MB") {
		base = MB
		fstr := strings.TrimSuffix(s, "MB")
		f, err = strconv.ParseFloat(fstr, 64)
		if err != nil {
			return err
		}
	} else if strings.HasSuffix(s, "KB") {
		base = KB
		fstr := strings.TrimSuffix(s, "KB")
		f, err = strconv.ParseFloat(fstr, 64)
		if err != nil {
			return err
		}
	} else {
		return errors.New("unit not support")
	}

	q.s = s
	q.i = int64(f * float64(base))
	return nil
}

func (q *BandwidthQuantity) UnmarshalJSON(b []byte) error {
	if len(b) == 4 && string(b) == "null" {
		return nil
	}

	var str string
	err := json.Unmarshal(b, &str)
	if err != nil {
		return err
	}

	return q.UnmarshalString(str)
}

func (q *BandwidthQuantity) MarshalJSON() ([]byte, error) {
	return []byte("\"" + q.s + "\""), nil
}

func (q *BandwidthQuantity) Bytes() int64 {
	return q.i
}
