package cert

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"time"
)

// Artifacts hosts a private key, its corresponding serving certificate and
// the CA certificate that signs the serving certificate.
type Artifacts struct {
	// PEM encoded private key
	Key []byte
	// PEM encoded serving certificate
	Cert []byte
	// PEM encoded CA private key
	CAKey []byte
	// PEM encoded CA certificate
	CACert []byte
	// Resource version of the certs
	ResourceVersion string
}

// Generator is an interface to provision the serving certificate.
type Generator interface {
	// Generate returns a Artifacts struct.
	Generate(CommonName string) (*Artifacts, error)
	// SetCA sets the PEM-encoded CA private key and CA cert for signing the generated serving cert.
	SetCA(caKey, caCert []byte)
}

// ValidCACert think cert and key are valid if they meet the following requirements:
// - key and cert are valid pair
// - caCert is the root ca of cert
// - cert is for dnsName
// - cert won't expire before time
func ValidCACert(key, cert, caCert []byte, dnsName string, time time.Time) bool {
	if len(key) == 0 || len(cert) == 0 || len(caCert) == 0 {
		return false
	}
	// Verify key and cert are valid pair
	_, err := tls.X509KeyPair(cert, key)
	if err != nil {
		return false
	}

	// Verify cert is valid for at least 1 year.
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caCert) {
		return false
	}
	block, _ := pem.Decode(cert)
	if block == nil {
		return false
	}
	c, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return false
	}
	ops := x509.VerifyOptions{
		DNSName:     dnsName,
		Roots:       pool,
		CurrentTime: time,
	}
	_, err = c.Verify(ops)
	return err == nil
}
