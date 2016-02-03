package pcrypto

import (
	"crypto/aes"
	"fmt"
	"testing"
)

func Test_Encrypto(t *testing.T) {
	pp := new(Pcrypto)
	pp.Init([]byte("Hana"))
	res, err := pp.Encrypto([]byte("Just One Test!"))
	if err != nil {
		t.Error(err)
	}

	fmt.Printf("[%x]\n", res)
}

func Test_Decrypto(t *testing.T) {
	pp := new(Pcrypto)
	pp.Init([]byte("Hana"))
	res, err := pp.Encrypto([]byte("Just One Test!"))
	if err != nil {
		t.Error(err)
	}

	res, err = pp.Decrypto(res)
	if err != nil {
		t.Error(err)
	}

	fmt.Printf("[%s]\n", string(res))
}

func Test_PKCS7Padding(t *testing.T) {
	ltt := []byte("Test_PKCS7Padding")
	ltt = PKCS7Padding(ltt, aes.BlockSize)
	fmt.Printf("[%x]\n", (ltt))
}

func Test_PKCS7UnPadding(t *testing.T) {
	ltt := []byte("Test_PKCS7Padding")
	ltt = PKCS7Padding(ltt, aes.BlockSize)
	ltt = PKCS7UnPadding(ltt)
	fmt.Printf("[%x]\n", ltt)
}
