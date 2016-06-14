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
	"bytes"
	"compress/gzip"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
)

type Pcrypto struct {
	pkey []byte
	paes cipher.Block
	// 0: nono; 1:compress; 2: encrypt; 3: compress and encrypt
	ptyp int
}

func (pc *Pcrypto) Init(key []byte, ptyp int) error {
	var err error
	pc.pkey = pKCS7Padding(key, aes.BlockSize)
	pc.paes, err = aes.NewCipher(pc.pkey)
	if ptyp == 1 || ptyp == 2 || ptyp == 3 {
		pc.ptyp = ptyp
	} else {
		pc.ptyp = 0
	}

	return err
}

func (pc *Pcrypto) Encrypt(src []byte) ([]byte, error) {
	var zbuf bytes.Buffer

	// gzip
	if pc.ptyp == 1 || pc.ptyp == 3 {
		zwr, err := gzip.NewWriterLevel(&zbuf, gzip.DefaultCompression)
		if err != nil {
			return nil, err
		}
		defer zwr.Close()
		zwr.Write(src)
		zwr.Flush()
		src = zbuf.Bytes()
	}

	// aes
	if pc.ptyp == 2 || pc.ptyp == 3 {
		src = pKCS7Padding(src, aes.BlockSize)
		blockMode := cipher.NewCBCEncrypter(pc.paes, pc.pkey)
		crypted := make([]byte, len(src))
		blockMode.CryptBlocks(crypted, src)
		src = crypted
	}

	return src, nil
}

func (pc *Pcrypto) Decrypt(str []byte) ([]byte, error) {
	// aes
	if pc.ptyp == 2 || pc.ptyp == 3 {
		decryptText, err := hex.DecodeString(fmt.Sprintf("%x", str))
		if err != nil {
			return nil, err
		}

		if len(decryptText)%aes.BlockSize != 0 {
			return nil, errors.New("crypto/cipher: ciphertext is not a multiple of the block size")
		}

		blockMode := cipher.NewCBCDecrypter(pc.paes, pc.pkey)

		blockMode.CryptBlocks(decryptText, decryptText)
		str = pKCS7UnPadding(decryptText)
	}

	// gunzip
	if pc.ptyp == 1 || pc.ptyp == 3 {
		zbuf := bytes.NewBuffer(str)
		zrd, err := gzip.NewReader(zbuf)
		if err != nil {
			return nil, err
		}
		defer zrd.Close()
		str, _ = ioutil.ReadAll(zrd)
	}

	return str, nil
}

func pKCS7Padding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}

func pKCS7UnPadding(origData []byte) []byte {
	length := len(origData)
	unpadding := int(origData[length-1])
	return origData[:(length - unpadding)]
}

func GetAuthKey(str string) (authKey string) {
	md5Ctx := md5.New()
	md5Ctx.Write([]byte(str))
	md5Str := md5Ctx.Sum(nil)
	return hex.EncodeToString(md5Str)
}
