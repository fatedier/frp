package auth

import (
	"crypto"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"

	"github.com/go-jose/go-jose/v4"
)

func DecodeJWKSFile(jwkBytes []byte) (*jose.JSONWebKeySet, error) {
	jwks := new(jose.JSONWebKeySet)
	err := json.Unmarshal(jwkBytes, jwks)
	if err != nil {
		return nil, err
	}
	return jwks, nil
}

func DecodePemCert(pemBytes []byte) ([]crypto.PublicKey, error) {
	keys := []crypto.PublicKey{}
	for len(pemBytes) > 0 {
		var block *pem.Block
		block, pemBytes = pem.Decode(pemBytes)
		if block == nil {
			break
		}
		switch block.Type {
		case "CERTIFICATE":
			cert, err := x509.ParseCertificate(block.Bytes)
			if err != nil {
				return nil, err
			}
			keys = append(keys, cert.PublicKey.(crypto.PublicKey))
		case "PUBLIC KEY":
			key, err := x509.ParsePKIXPublicKey(block.Bytes)
			if err != nil {
				return nil, err
			}
			keys = append(keys, key)
		default:
			continue
		}
	}
	if len(keys) == 0 {
		return nil, errors.New("data does not contain any valid keys")
	}
	return keys, nil
}

func DecodeJWKS(jwks *jose.JSONWebKeySet) []crypto.PublicKey {
	keys := []crypto.PublicKey{}
	for _, k := range jwks.Keys {
		keys = append(keys, k.Key.(crypto.PublicKey))
	}
	return keys
}
