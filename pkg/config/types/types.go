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

package types

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

const (
	MB = 1024 * 1024
	KB = 1024

	BandwidthLimitModeClient = "client"
	BandwidthLimitModeServer = "server"
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
	switch {
	case strings.HasSuffix(s, "MB"):
		base = MB
		fstr := strings.TrimSuffix(s, "MB")
		f, err = strconv.ParseFloat(fstr, 64)
		if err != nil {
			return err
		}
	case strings.HasSuffix(s, "KB"):
		base = KB
		fstr := strings.TrimSuffix(s, "KB")
		f, err = strconv.ParseFloat(fstr, 64)
		if err != nil {
			return err
		}
	default:
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

type PortsRange struct {
	Start  int `json:"start,omitempty"`
	End    int `json:"end,omitempty"`
	Single int `json:"single,omitempty"`
}

type PortsRangeSlice []PortsRange

func (p PortsRangeSlice) String() string {
	if len(p) == 0 {
		return ""
	}
	strs := []string{}
	for _, v := range p {
		if v.Single > 0 {
			strs = append(strs, strconv.Itoa(v.Single))
		} else {
			strs = append(strs, strconv.Itoa(v.Start)+"-"+strconv.Itoa(v.End))
		}
	}
	return strings.Join(strs, ",")
}

// the format of str is like "1000-2000,3000,4000-5000"
func NewPortsRangeSliceFromString(str string) ([]PortsRange, error) {
	str = strings.TrimSpace(str)
	out := []PortsRange{}
	numRanges := strings.Split(str, ",")
	for _, numRangeStr := range numRanges {
		// 1000-2000 or 2001
		numArray := strings.Split(numRangeStr, "-")
		// length: only 1 or 2 is correct
		rangeType := len(numArray)
		switch rangeType {
		case 1:
			// single number
			singleNum, err := strconv.ParseInt(strings.TrimSpace(numArray[0]), 10, 64)
			if err != nil {
				return nil, fmt.Errorf("range number is invalid, %v", err)
			}
			out = append(out, PortsRange{Single: int(singleNum)})
		case 2:
			// range numbers
			minNum, err := strconv.ParseInt(strings.TrimSpace(numArray[0]), 10, 64)
			if err != nil {
				return nil, fmt.Errorf("range number is invalid, %v", err)
			}
			maxNum, err := strconv.ParseInt(strings.TrimSpace(numArray[1]), 10, 64)
			if err != nil {
				return nil, fmt.Errorf("range number is invalid, %v", err)
			}
			if maxNum < minNum {
				return nil, fmt.Errorf("range number is invalid")
			}
			out = append(out, PortsRange{Start: int(minNum), End: int(maxNum)})
		default:
			return nil, fmt.Errorf("range number is invalid")
		}
	}
	return out, nil
}
