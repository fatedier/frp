// Copyright 2017 fatedier, fatedier@gmail.com
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

package util

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
)

// RandId return a rand string used in frp.
func RandId() (id string, err error) {
	return RandIdWithLen(8)
}

// RandIdWithLen return a rand string with idLen length.
func RandIdWithLen(idLen int) (id string, err error) {
	b := make([]byte, idLen)
	_, err = rand.Read(b)
	if err != nil {
		return
	}

	id = fmt.Sprintf("%x", b)
	return
}

func GetAuthKey(token string, timestamp int64) (key string) {
	token = token + fmt.Sprintf("%d", timestamp)
	md5Ctx := md5.New()
	md5Ctx.Write([]byte(token))
	data := md5Ctx.Sum(nil)
	return hex.EncodeToString(data)
}

func CanonicalAddr(host string, port int) (addr string) {
	if port == 80 || port == 443 {
		addr = host
	} else {
		addr = fmt.Sprintf("%s:%d", host, port)
	}
	return
}

func ParseRangeNumbers(rangeStr string) (numbers []int64, err error) {
	rangeStr = strings.TrimSpace(rangeStr)
	numbers = make([]int64, 0)
	// e.g. 1000-2000,2001,2002,3000-4000
	numRanges := strings.Split(rangeStr, ",")
	for _, numRangeStr := range numRanges {
		// 1000-2000 or 2001
		numArray := strings.Split(numRangeStr, "-")
		// length: only 1 or 2 is correct
		rangeType := len(numArray)
		if rangeType == 1 {
			// single number
			singleNum, errRet := strconv.ParseInt(strings.TrimSpace(numArray[0]), 10, 64)
			if errRet != nil {
				err = fmt.Errorf("range number is invalid, %v", errRet)
				return
			}
			numbers = append(numbers, singleNum)
		} else if rangeType == 2 {
			// range numbers
			min, errRet := strconv.ParseInt(strings.TrimSpace(numArray[0]), 10, 64)
			if errRet != nil {
				err = fmt.Errorf("range number is invalid, %v", errRet)
				return
			}
			max, errRet := strconv.ParseInt(strings.TrimSpace(numArray[1]), 10, 64)
			if errRet != nil {
				err = fmt.Errorf("range number is invalid, %v", errRet)
				return
			}
			if max < min {
				err = fmt.Errorf("range number is invalid")
				return
			}
			for i := min; i <= max; i++ {
				numbers = append(numbers, i)
			}
		} else {
			err = fmt.Errorf("range number is invalid")
			return
		}
	}
	return
}

func GenerateResponseErrorString(summary string, err error, detailed bool) string {
	if detailed {
		return err.Error()
	} else {
		return summary
	}
}
