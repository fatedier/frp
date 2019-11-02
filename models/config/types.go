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
	"errors"
	"strconv"
	"strings"
)

const (
	MB = 1024 * 1024
	KB = 1024
)

type BandwithQuantity struct {
	s string // MB or KB

	i int64 // bytes
}

func NewBandwithQuantity(s string) (BandwithQuantity, error) {
	q := BandwithQuantity{}
	err := q.UnmarshalString(s)
	if err != nil {
		return q, err
	}
	return q, nil
}

func (q *BandwithQuantity) Equal(u *BandwithQuantity) bool {
	if q == nil && u == nil {
		return true
	}
	if q != nil && u != nil {
		return q.i == u.i
	}
	return false
}

func (q *BandwithQuantity) String() string {
	return q.s
}

func (q *BandwithQuantity) UnmarshalString(s string) error {
	q.s = strings.TrimSpace(s)
	if q.s == "" {
		return nil
	}

	var (
		base int64
		f    float64
		err  error
	)
	if strings.HasSuffix(s, "MB") {
		base = MB
		s = strings.TrimSuffix(s, "MB")
		f, err = strconv.ParseFloat(s, 64)
		if err != nil {
			return err
		}
	} else if strings.HasSuffix(s, "KB") {
		base = KB
		s = strings.TrimSuffix(s, "KB")
		f, err = strconv.ParseFloat(s, 64)
		if err != nil {
			return err
		}
	} else {
		return errors.New("unit not support")
	}

	q.i = int64(f * float64(base))
	return nil
}

func (q *BandwithQuantity) UnmarshalJSON(b []byte) error {
	return q.UnmarshalString(string(b))
}

func (q *BandwithQuantity) MarshalJSON() ([]byte, error) {
	return []byte(q.s), nil
}

func (q *BandwithQuantity) Bytes() int64 {
	return q.i
}
