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
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
)

type Pcrypto struct {
	pkey []byte
	paes cipher.Block
}

func (pc *Pcrypto) Init(key []byte) error {
	var err error
	pc.pkey = pkKeyPadding(key)
	pc.paes, err = aes.NewCipher(pc.pkey)
	return err
}

func (pc *Pcrypto) Encrypt(src []byte) ([]byte, error) {
	// aes
	src = pKCS5Padding(src, aes.BlockSize)
	ciphertext := make([]byte, aes.BlockSize+len(src))

	// The IV needs to be unique, but not secure. Therefore it's common to
	// include it at the beginning of the ciphertext.
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}
	blockMode := cipher.NewCBCEncrypter(pc.paes, iv)
	blockMode.CryptBlocks(ciphertext[aes.BlockSize:], src)
	return ciphertext, nil
}

func (pc *Pcrypto) Decrypt(str []byte) ([]byte, error) {
	// aes
	ciphertext, err := hex.DecodeString(fmt.Sprintf("%x", str))
	if err != nil {
		return nil, err
	}

	if len(ciphertext) < aes.BlockSize {
		return nil, fmt.Errorf("ciphertext too short")
	}
	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	if len(ciphertext)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("crypto/cipher: ciphertext is not a multiple of the block size")
	}

	blockMode := cipher.NewCBCDecrypter(pc.paes, iv)
	blockMode.CryptBlocks(ciphertext, ciphertext)
	return pKCS5UnPadding(ciphertext), nil
}

func (pc *Pcrypto) Compression(src []byte) ([]byte, error) {
	var zbuf bytes.Buffer
	zwr, err := gzip.NewWriterLevel(&zbuf, gzip.DefaultCompression)
	if err != nil {
		return nil, err
	}
	defer zwr.Close()
	zwr.Write(src)
	zwr.Flush()
	return zbuf.Bytes(), nil
}

func (pc *Pcrypto) Decompression(src []byte) ([]byte, error) {
	zbuf := bytes.NewBuffer(src)
	zrd, err := gzip.NewReader(zbuf)
	if err != nil {
		return nil, err
	}
	defer zrd.Close()
	str, _ := ioutil.ReadAll(zrd)
	return str, nil
}

func pkKeyPadding(key []byte) []byte {
	l := len(key)
	if l == 16 || l == 24 || l == 32 {
		return key
	}
	if l < 16 {
		return append(key, bytes.Repeat([]byte{byte(0)}, 16-l)...)
	} else if l < 24 {
		return append(key, bytes.Repeat([]byte{byte(0)}, 24-l)...)
	} else if l < 32 {
		return append(key, bytes.Repeat([]byte{byte(0)}, 32-l)...)
	} else {
		md5Ctx := md5.New()
		md5Ctx.Write(key)
		md5Str := md5Ctx.Sum(nil)
		return []byte(hex.EncodeToString(md5Str))
	}
}

func pKCS5Padding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}

func pKCS5UnPadding(origData []byte) []byte {
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
