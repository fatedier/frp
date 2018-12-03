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
	"crypto/rand"
	"crypto/sha1"
	"io"

	"golang.org/x/crypto/pbkdf2"
)

var (
	DefaultSalt = "crypto"
)

// NewWriter returns a new Writer that encrypts bytes to w.
func NewWriter(w io.Writer, key []byte) (*Writer, error) {
	key = pbkdf2.Key(key, []byte(DefaultSalt), 64, aes.BlockSize, sha1.New)

	// random iv
	iv := make([]byte, aes.BlockSize)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	return &Writer{
		w: w,
		enc: &cipher.StreamWriter{
			S: cipher.NewCFBEncrypter(block, iv),
			W: w,
		},
		key: key,
		iv:  iv,
	}, nil
}

// Writer is an io.Writer that can write encrypted bytes.
// Now it only support aes-128-cfb.
type Writer struct {
	w      io.Writer
	enc    *cipher.StreamWriter
	key    []byte
	iv     []byte
	ivSend bool
	err    error
}

// Write satisfies the io.Writer interface.
func (w *Writer) Write(p []byte) (nRet int, errRet error) {
	if w.err != nil {
		return 0, w.err
	}

	// When write is first called, iv will be written to w.w
	if !w.ivSend {
		w.ivSend = true
		_, errRet = w.w.Write(w.iv)
		if errRet != nil {
			w.err = errRet
			return
		}
	}

	nRet, errRet = w.enc.Write(p)
	if errRet != nil {
		w.err = errRet
	}
	return
}
