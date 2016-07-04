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

package pcrypto

import (
	"testing"
)

var (
	pp *Pcrypto
)

func init() {
	pp = &Pcrypto{}
	pp.Init([]byte("Hana"))
}

func TestEncrypt(t *testing.T) {
	testStr := "Test Encrypt!"
	res, err := pp.Encrypt([]byte(testStr))
	if err != nil {
		t.Fatalf("encrypt error: %v", err)
	}

	res, err = pp.Decrypt([]byte(res))
	if err != nil {
		t.Fatalf("decrypt error: %v", err)
	}

	if string(res) != testStr {
		t.Fatalf("test encrypt error, from [%s] to [%s]", testStr, string(res))
	}
}

func TestCompression(t *testing.T) {
	testStr := "Test Compression!"
	res, err := pp.Compression([]byte(testStr))
	if err != nil {
		t.Fatalf("compression error: %v", err)
	}

	res, err = pp.Decompression(res)
	if err != nil {
		t.Fatalf("decompression error: %v", err)
	}

	if string(res) != testStr {
		t.Fatalf("test compression error, from [%s] to [%s]", testStr, string(res))
	}
}
