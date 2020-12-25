// Copyright 2019 fatedier, fatedier@gmail.com
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
	"encoding/json"
	"errors"
	"strconv"
	"strings"
)

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
