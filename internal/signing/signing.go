package signing

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"strings"
)

type Signer struct {
	privateKey *rsa.PrivateKey
}

func NewSigner(privateKeySource string) (*Signer, error) {
	if privateKeySource == "" {
		return nil, nil
	}

	var pemData []byte
	if strings.Contains(privateKeySource, "BEGIN") {
		pemData = []byte(privateKeySource)
	} else {
		var err error
		pemData, err = os.ReadFile(privateKeySource)
		if err != nil {
			return nil, err
		}
	}

	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, errors.New("failed to decode PEM block")
	}

	var privateKey *rsa.PrivateKey

	switch block.Type {
	case "RSA PRIVATE KEY":
		var err error
		privateKey, err = x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return nil, err
		}

	case "PRIVATE KEY":
		parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, err
		}

		key, ok := parsed.(*rsa.PrivateKey)
		if !ok {
			return nil, errors.New("failed to parse RSA private key")
		}
		privateKey = key

	default:
		return nil, errors.New("unsupported key type")
	}

	return &Signer{
		privateKey: privateKey,
	}, nil
}

func (s *Signer) Sign(data []byte) (string, error) {
	hash := sha256.Sum256(data)
	signature, err := rsa.SignPKCS1v15(rand.Reader, s.privateKey, crypto.SHA256, hash[:])
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(signature), nil
}

func FormatSignatureHeader(signature string) string {
	return fmt.Sprintf("sig=:%s:, keyid=\"main\"", signature)
}
