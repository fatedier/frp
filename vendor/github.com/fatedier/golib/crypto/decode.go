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

package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha1"
	"io"

	"golang.org/x/crypto/pbkdf2"
)

// NewReader returns a new Reader that decrypts bytes from r
func NewReader(r io.Reader, key []byte) *Reader {
	key = pbkdf2.Key(key, []byte(DefaultSalt), 64, aes.BlockSize, sha1.New)

	return &Reader{
		r:   r,
		key: key,
	}
}

// Reader is an io.Reader that can read encrypted bytes.
// Now it only supports aes-128-cfb.
type Reader struct {
	r   io.Reader
	dec *cipher.StreamReader
	key []byte
	iv  []byte
	err error
}

// Read satisfies the io.Reader interface.
func (r *Reader) Read(p []byte) (nRet int, errRet error) {
	if r.err != nil {
		return 0, r.err
	}

	if r.dec == nil {
		iv := make([]byte, aes.BlockSize)
		if _, errRet = io.ReadFull(r.r, iv); errRet != nil {
			return
		}
		r.iv = iv

		block, err := aes.NewCipher(r.key)
		if err != nil {
			errRet = err
			return
		}
		r.dec = &cipher.StreamReader{
			S: cipher.NewCFBDecrypter(block, iv),
			R: r.r,
		}
	}

	nRet, errRet = r.dec.Read(p)
	if errRet != nil {
		r.err = errRet
	}
	return
}
