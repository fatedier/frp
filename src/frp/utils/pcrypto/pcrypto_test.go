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
	"fmt"
	"testing"
)

func TestEncrypt(t *testing.T) {
	return
	pp := new(Pcrypto)
	pp.Init([]byte("Hana"), 1)
	res, err := pp.Encrypt([]byte("Test Encrypt!"))
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("Encrypt: len %d, [%x]\n", len(res), res)
}

func TestDecrypt(t *testing.T) {
	fmt.Println("*****************************************************")
	{
		pp := new(Pcrypto)
		pp.Init([]byte("Hana"), 0)
		res, err := pp.Encrypt([]byte("Test Decrypt! 0"))
		if err != nil {
			t.Fatal(err)
		}

		res, err = pp.Decrypt(res)
		if err != nil {
			t.Fatal(err)
		}

		fmt.Printf("[%s]\n", string(res))
	}
	{
		pp := new(Pcrypto)
		pp.Init([]byte("Hana"), 1)
		res, err := pp.Encrypt([]byte("Test Decrypt! 1"))
		if err != nil {
			t.Fatal(err)
		}

		res, err = pp.Decrypt(res)
		if err != nil {
			t.Fatal(err)
		}

		fmt.Printf("[%s]\n", string(res))
	}
	{
		pp := new(Pcrypto)
		pp.Init([]byte("Hana"), 2)
		res, err := pp.Encrypt([]byte("Test Decrypt! 2"))
		if err != nil {
			t.Fatal(err)
		}

		res, err = pp.Decrypt(res)
		if err != nil {
			t.Fatal(err)
		}

		fmt.Printf("[%s]\n", string(res))
	}
	{
		pp := new(Pcrypto)
		pp.Init([]byte("Hana"), 3)
		res, err := pp.Encrypt([]byte("Test Decrypt! 3"))
		if err != nil {
			t.Fatal(err)
		}

		res, err = pp.Decrypt(res)
		if err != nil {
			t.Fatal(err)
		}

		fmt.Printf("[%s]\n", string(res))
	}

}
