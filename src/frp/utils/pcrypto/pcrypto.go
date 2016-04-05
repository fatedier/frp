package pcrypto

import (
	"bytes"
	"compress/gzip"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
)

type Pcrypto struct {
	pkey []byte
	paes cipher.Block
}

func (pc *Pcrypto) Init(key []byte) error {
	var err error
	pc.pkey = pKCS7Padding(key, aes.BlockSize)
	pc.paes, err = aes.NewCipher(pc.pkey)

	return err
}

func (pc *Pcrypto) Encrypt(src []byte) ([]byte, error) {
	// gzip
	var zbuf bytes.Buffer
	zwr, err := gzip.NewWriterLevel(&zbuf, -1)
	if err != nil {
		return nil, err
	}
	defer zwr.Close()
	zwr.Write(src)
	zwr.Flush()

	// aes
	src = pKCS7Padding(zbuf.Bytes(), aes.BlockSize)
	blockMode := cipher.NewCBCEncrypter(pc.paes, pc.pkey)
	crypted := make([]byte, len(src))
	blockMode.CryptBlocks(crypted, src)

	// base64
	return []byte(base64.StdEncoding.EncodeToString(crypted)), nil
}

func (pc *Pcrypto) Decrypt(str []byte) ([]byte, error) {
	// base64
	data, err := base64.StdEncoding.DecodeString(string(str))
	if err != nil {
		return nil, err
	}

	// aes
	decryptText, err := hex.DecodeString(fmt.Sprintf("%x", data))
	if err != nil {
		return nil, err
	}

	if len(decryptText)%aes.BlockSize != 0 {
		return nil, errors.New("crypto/cipher: ciphertext is not a multiple of the block size")
	}

	blockMode := cipher.NewCBCDecrypter(pc.paes, pc.pkey)

	blockMode.CryptBlocks(decryptText, decryptText)
	decryptText = pKCS7UnPadding(decryptText)

	// gunzip
	zbuf := bytes.NewBuffer(decryptText)
	zrd, _ := gzip.NewReader(zbuf)
	defer zrd.Close()
	data, _ = ioutil.ReadAll(zrd)

	return data, nil
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
