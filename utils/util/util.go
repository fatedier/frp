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
