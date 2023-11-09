package v1

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"os"
	"path/filepath"
)

const (
	// custom define
	SSHClientLoginUserPrefix = "_frpc_ssh_client_"
)

// encodePrivateKeyToPEM encodes Private Key from RSA to PEM format
func GeneratePrivateKey() ([]byte, error) {
	privateKey, err := generatePrivateKey()
	if err != nil {
		return nil, errors.New("gen private key error")
	}

	privBlock := pem.Block{
		Type:    "RSA PRIVATE KEY",
		Headers: nil,
		Bytes:   x509.MarshalPKCS1PrivateKey(privateKey),
	}

	return pem.EncodeToMemory(&privBlock), nil
}

// generatePrivateKey creates a RSA Private Key of specified byte size
func generatePrivateKey() (*rsa.PrivateKey, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, err
	}

	err = privateKey.Validate()
	if err != nil {
		return nil, err
	}
	return privateKey, nil
}

func LoadFilesInDirectory(dirPath string) (map[string]string, error) {
	fileMap := make(map[string]string)

	files, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		filename := file.Name()
		filePath := filepath.Join(dirPath, filename)

		content, err := os.ReadFile(filePath)
		if err != nil {
			return nil, err
		}

		fileMap[filename] = string(content)
	}

	return fileMap, nil
}
