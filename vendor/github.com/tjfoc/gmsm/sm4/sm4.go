/*
Copyright Suzhou Tongji Fintech Research Institute 2017 All Rights Reserved.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

                 http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package sm4

import (
	"crypto/cipher"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"io/ioutil"
	"os"
	"strconv"
)

const BlockSize = 16

type SM4Key []byte

type KeySizeError int

// Cipher is an instance of SM4 encryption.
type Sm4Cipher struct {
	subkeys []uint32
	block1  []uint32
	block2  []byte
}

// sm4密钥参量
var fk = [4]uint32{
	0xa3b1bac6, 0x56aa3350, 0x677d9197, 0xb27022dc,
}

// sm4密钥参量
var ck = [32]uint32{
	0x00070e15, 0x1c232a31, 0x383f464d, 0x545b6269,
	0x70777e85, 0x8c939aa1, 0xa8afb6bd, 0xc4cbd2d9,
	0xe0e7eef5, 0xfc030a11, 0x181f262d, 0x343b4249,
	0x50575e65, 0x6c737a81, 0x888f969d, 0xa4abb2b9,
	0xc0c7ced5, 0xdce3eaf1, 0xf8ff060d, 0x141b2229,
	0x30373e45, 0x4c535a61, 0x686f767d, 0x848b9299,
	0xa0a7aeb5, 0xbcc3cad1, 0xd8dfe6ed, 0xf4fb0209,
	0x10171e25, 0x2c333a41, 0x484f565d, 0x646b7279,
}

// sm4密钥参量
var sbox = [256]uint8{
	0xd6, 0x90, 0xe9, 0xfe, 0xcc, 0xe1, 0x3d, 0xb7, 0x16, 0xb6, 0x14, 0xc2, 0x28, 0xfb, 0x2c, 0x05,
	0x2b, 0x67, 0x9a, 0x76, 0x2a, 0xbe, 0x04, 0xc3, 0xaa, 0x44, 0x13, 0x26, 0x49, 0x86, 0x06, 0x99,
	0x9c, 0x42, 0x50, 0xf4, 0x91, 0xef, 0x98, 0x7a, 0x33, 0x54, 0x0b, 0x43, 0xed, 0xcf, 0xac, 0x62,
	0xe4, 0xb3, 0x1c, 0xa9, 0xc9, 0x08, 0xe8, 0x95, 0x80, 0xdf, 0x94, 0xfa, 0x75, 0x8f, 0x3f, 0xa6,
	0x47, 0x07, 0xa7, 0xfc, 0xf3, 0x73, 0x17, 0xba, 0x83, 0x59, 0x3c, 0x19, 0xe6, 0x85, 0x4f, 0xa8,
	0x68, 0x6b, 0x81, 0xb2, 0x71, 0x64, 0xda, 0x8b, 0xf8, 0xeb, 0x0f, 0x4b, 0x70, 0x56, 0x9d, 0x35,
	0x1e, 0x24, 0x0e, 0x5e, 0x63, 0x58, 0xd1, 0xa2, 0x25, 0x22, 0x7c, 0x3b, 0x01, 0x21, 0x78, 0x87,
	0xd4, 0x00, 0x46, 0x57, 0x9f, 0xd3, 0x27, 0x52, 0x4c, 0x36, 0x02, 0xe7, 0xa0, 0xc4, 0xc8, 0x9e,
	0xea, 0xbf, 0x8a, 0xd2, 0x40, 0xc7, 0x38, 0xb5, 0xa3, 0xf7, 0xf2, 0xce, 0xf9, 0x61, 0x15, 0xa1,
	0xe0, 0xae, 0x5d, 0xa4, 0x9b, 0x34, 0x1a, 0x55, 0xad, 0x93, 0x32, 0x30, 0xf5, 0x8c, 0xb1, 0xe3,
	0x1d, 0xf6, 0xe2, 0x2e, 0x82, 0x66, 0xca, 0x60, 0xc0, 0x29, 0x23, 0xab, 0x0d, 0x53, 0x4e, 0x6f,
	0xd5, 0xdb, 0x37, 0x45, 0xde, 0xfd, 0x8e, 0x2f, 0x03, 0xff, 0x6a, 0x72, 0x6d, 0x6c, 0x5b, 0x51,
	0x8d, 0x1b, 0xaf, 0x92, 0xbb, 0xdd, 0xbc, 0x7f, 0x11, 0xd9, 0x5c, 0x41, 0x1f, 0x10, 0x5a, 0xd8,
	0x0a, 0xc1, 0x31, 0x88, 0xa5, 0xcd, 0x7b, 0xbd, 0x2d, 0x74, 0xd0, 0x12, 0xb8, 0xe5, 0xb4, 0xb0,
	0x89, 0x69, 0x97, 0x4a, 0x0c, 0x96, 0x77, 0x7e, 0x65, 0xb9, 0xf1, 0x09, 0xc5, 0x6e, 0xc6, 0x84,
	0x18, 0xf0, 0x7d, 0xec, 0x3a, 0xdc, 0x4d, 0x20, 0x79, 0xee, 0x5f, 0x3e, 0xd7, 0xcb, 0x39, 0x48,
}

func rl(x uint32, i uint8) uint32 { return (x << (i % 32)) | (x >> (32 - (i % 32))) }

func l0(b uint32) uint32 { return b ^ rl(b, 13) ^ rl(b, 23) }

func l1(b uint32) uint32 { return b ^ rl(b, 2) ^ rl(b, 10) ^ rl(b, 18) ^ rl(b, 24) }

func feistel0(x0, x1, x2, x3, rk uint32) uint32 { return x0 ^ l0(p(x1^x2^x3^rk)) }

func feistel1(x0, x1, x2, x3, rk uint32) uint32 { return x0 ^ l1(p(x1^x2^x3^rk)) }

//非线性变换τ(.)
func p(a uint32) uint32 {
	return (uint32(sbox[a>>24]) << 24) ^ (uint32(sbox[(a>>16)&0xff]) << 16) ^ (uint32(sbox[(a>>8)&0xff]) << 8) ^ uint32(sbox[(a)&0xff])
}

/*
func permuteInitialBlock(block []byte) []uint32 {
	b := make([]uint32, 4, 4)
	for i := 0; i < 4; i++ {
		b[i] = (uint32(block[i*4]) << 24) | (uint32(block[i*4+1]) << 16) |
			(uint32(block[i*4+2]) << 8) | (uint32(block[i*4+3]))
	}
	return b
}

func permuteFinalBlock(block []uint32) []byte {
	b := make([]byte, 16, 16)
	for i := 0; i < 4; i++ {
		b[i*4] = uint8(block[i] >> 24)
		b[i*4+1] = uint8(block[i] >> 16)
		b[i*4+2] = uint8(block[i] >> 8)
		b[i*4+3] = uint8(block[i])
	}
	return b
}

func cryptBlock(subkeys []uint32, dst, src []byte, decrypt bool) {
	var tm uint32
	b := permuteInitialBlock(src)
	for i := 0; i < 32; i++ {
		if decrypt {
			tm = feistel1(b[0], b[1], b[2], b[3], subkeys[31-i])
		} else {
			tm = feistel1(b[0], b[1], b[2], b[3], subkeys[i])
		}
		b[0], b[1], b[2], b[3] = b[1], b[2], b[3], tm
	}
	b[0], b[1], b[2], b[3] = b[3], b[2], b[1], b[0]
	copy(dst, permuteFinalBlock(b))
}
*/

func permuteInitialBlock(b []uint32, block []byte) {
	for i := 0; i < 4; i++ {
		b[i] = (uint32(block[i*4]) << 24) | (uint32(block[i*4+1]) << 16) |
			(uint32(block[i*4+2]) << 8) | (uint32(block[i*4+3]))
	}
}

func permuteFinalBlock(b []byte, block []uint32) {
	for i := 0; i < 4; i++ {
		b[i*4] = uint8(block[i] >> 24)
		b[i*4+1] = uint8(block[i] >> 16)
		b[i*4+2] = uint8(block[i] >> 8)
		b[i*4+3] = uint8(block[i])
	}
}
func cryptBlock(subkeys []uint32, b []uint32, r []byte, dst, src []byte, decrypt bool) {
	var tm uint32

	permuteInitialBlock(b, src)
	for i := 0; i < 32; i++ {
		if decrypt {
			tm = b[0] ^ l1(p(b[1]^b[2]^b[3]^subkeys[31 - i]))
			//			tm = feistel1(b[0], b[1], b[2], b[3], subkeys[31-i])
		} else {
			tm = b[0] ^ l1(p(b[1]^b[2]^b[3]^subkeys[i]))
			//	tm = feistel1(b[0], b[1], b[2], b[3], subkeys[i])
		}
		b[0], b[1], b[2], b[3] = b[1], b[2], b[3], tm
	}
	b[0], b[1], b[2], b[3] = b[3], b[2], b[1], b[0]
	permuteFinalBlock(r, b)
	copy(dst, r)
}

func generateSubKeys(key []byte) []uint32 {
	subkeys := make([]uint32, 32)
	b := make([]uint32, 4)
	//	b := permuteInitialBlock(key)
	permuteInitialBlock(b, key)
	b[0] ^= fk[0]
	b[1] ^= fk[1]
	b[2] ^= fk[2]
	b[3] ^= fk[3]
	for i := 0; i < 32; i++ {
		subkeys[i] = feistel0(b[0], b[1], b[2], b[3], ck[i])
		b[0], b[1], b[2], b[3] = b[1], b[2], b[3], subkeys[i]
	}
	return subkeys
}

func EncryptBlock(key SM4Key, dst, src []byte) {
	subkeys := generateSubKeys(key)
	cryptBlock(subkeys, make([]uint32, 4), make([]byte, 16), dst, src, false)
}

func DecryptBlock(key SM4Key, dst, src []byte) {
	subkeys := generateSubKeys(key)
	cryptBlock(subkeys, make([]uint32, 4), make([]byte, 16), dst, src, true)
}

func ReadKeyFromMem(data []byte, pwd []byte) (SM4Key, error) {
	block, _ := pem.Decode(data)
	if x509.IsEncryptedPEMBlock(block) {
		if block.Type != "SM4 ENCRYPTED KEY" {
			return nil, errors.New("SM4: unknown type")
		}
		if pwd == nil {
			return nil, errors.New("SM4: need passwd")
		}
		data, err := x509.DecryptPEMBlock(block, pwd)
		if err != nil {
			return nil, err
		}
		return data, nil
	}
	if block.Type != "SM4 KEY" {
		return nil, errors.New("SM4: unknown type")
	}
	return block.Bytes, nil
}

func ReadKeyFromPem(FileName string, pwd []byte) (SM4Key, error) {
	data, err := ioutil.ReadFile(FileName)
	if err != nil {
		return nil, err
	}
	return ReadKeyFromMem(data, pwd)
}

func WriteKeytoMem(key SM4Key, pwd []byte) ([]byte, error) {
	if pwd != nil {
		block, err := x509.EncryptPEMBlock(rand.Reader,
			"SM4 ENCRYPTED KEY", key, pwd, x509.PEMCipherAES256)
		if err != nil {
			return nil, err
		}
		return pem.EncodeToMemory(block), nil
	} else {
		block := &pem.Block{
			Type:  "SM4 KEY",
			Bytes: key,
		}
		return pem.EncodeToMemory(block), nil
	}
}

func WriteKeyToPem(FileName string, key SM4Key, pwd []byte) (bool, error) {
	var block *pem.Block

	if pwd != nil {
		var err error
		block, err = x509.EncryptPEMBlock(rand.Reader,
			"SM4 ENCRYPTED KEY", key, pwd, x509.PEMCipherAES256)
		if err != nil {
			return false, err
		}
	} else {
		block = &pem.Block{
			Type:  "SM4 KEY",
			Bytes: key,
		}
	}
	file, err := os.Create(FileName)
	if err != nil {
		return false, err
	}
	defer file.Close()
	err = pem.Encode(file, block)
	if err != nil {
		return false, nil
	}
	return true, nil
}

func (k KeySizeError) Error() string {
	return "SM4: invalid key size " + strconv.Itoa(int(k))
}

// NewCipher creates and returns a new cipher.Block.
func NewCipher(key []byte) (cipher.Block, error) {
	if len(key) != BlockSize {
		return nil, KeySizeError(len(key))
	}
	c := new(Sm4Cipher)
	c.subkeys = generateSubKeys(key)
	c.block1 = make([]uint32, 4)
	c.block2 = make([]byte, 16)
	return c, nil
}

func (c *Sm4Cipher) BlockSize() int {
	return BlockSize
}

func (c *Sm4Cipher) Encrypt(dst, src []byte) {
	cryptBlock(c.subkeys, c.block1, c.block2, dst, src, false)
}

func (c *Sm4Cipher) Decrypt(dst, src []byte) {
	cryptBlock(c.subkeys, c.block1, c.block2, dst, src, true)
}
