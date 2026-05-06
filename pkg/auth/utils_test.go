package auth

import (
	"crypto/rsa"
	"embed"
	"testing"

	"github.com/stretchr/testify/require"
)

//go:embed testSample/*.json
var jwksContent embed.FS

//go:embed testSample/*.pem
var pemContent embed.FS

//go:embed testSample/pki/*.pem
var pkiContent embed.FS

func TestJWKSDecodeFileSingle(t *testing.T) {
	r := require.New(t)
	data, _ := jwksContent.ReadFile("testSample/jwks_single.json")
	jwks, err := DecodeJWKSFile(data)
	r.NoError(err)
	r.Len(jwks.Keys, 1)
	r.Len(jwks.Key("00000000-0000-0000-0000-000000000000"), 1)
	r.Equal("RS256", jwks.Key("00000000-0000-0000-0000-000000000000")[0].Algorithm)
	r.Equal("sig", jwks.Key("00000000-0000-0000-0000-000000000000")[0].Use)
	r.Equal("00000000-0000-0000-0000-000000000000", jwks.Key("00000000-0000-0000-0000-000000000000")[0].KeyID)
	rsaK := jwks.Key("00000000-0000-0000-0000-000000000000")[0].Key.(*rsa.PublicKey)
	r.Equal(65_537, rsaK.E)
	r.Equal(128, rsaK.Size())
}

func TestJWKSDecodeFileMultiple(t *testing.T) {
	r := require.New(t)
	data, _ := jwksContent.ReadFile("testSample/jwks_multiple.json")
	jwks, err := DecodeJWKSFile(data)
	r.NoError(err)
	r.Len(jwks.Keys, 2)
	r.Len(jwks.Key("00000000-0000-0000-0000-000000000000"), 1)
	r.Len(jwks.Key("00000000-0000-0000-0000-000000000001"), 1)
}

func TestPemDecodeFileSingle(t *testing.T) {
	r := require.New(t)
	data, _ := pemContent.ReadFile("testSample/pem_single.pem")
	pems, err := DecodePemCert(data)
	r.NoError(err)
	r.Len(pems, 1)
	rsaK := pems[0].(*rsa.PublicKey)
	r.Equal(65_537, rsaK.E)
	r.Equal(128, rsaK.Size())
}

func TestPemDecodeFileMultiple(t *testing.T) {
	r := require.New(t)
	data, _ := pemContent.ReadFile("testSample/pem_multiple.pem")
	pems, err := DecodePemCert(data)
	r.NoError(err)
	r.Len(pems, 2)
}

func TestCertPemDecodeFileCa(t *testing.T) {
	r := require.New(t)
	data, _ := pkiContent.ReadFile("testSample/pki/ca.pem")
	pems, err := DecodePemCert(data)
	r.NoError(err)
	r.Len(pems, 1)
}

func TestCertPemDecodeFileServer(t *testing.T) {
	r := require.New(t)
	data, _ := pkiContent.ReadFile("testSample/pki/server.full.pem")
	pems, err := DecodePemCert(data)
	r.NoError(err)
	r.Len(pems, 2)
}
