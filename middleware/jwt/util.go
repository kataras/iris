package jwt

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
)

// ErrNotPEM is a panic error of the MustParseXXX functions when the data are not PEM-encoded.
var ErrNotPEM = errors.New("key must be PEM encoded")

// MustParseRSAPrivateKey encodes a PEM-encoded PKCS1 or PKCS8 private key protected with a password.
func MustParseRSAPrivateKey(key, password []byte) *rsa.PrivateKey {
	block, _ := pem.Decode(key)
	if block == nil {
		panic(ErrNotPEM)
	}

	var (
		parsedKey interface{}
		err       error
	)

	var blockDecrypted []byte
	if blockDecrypted, err = x509.DecryptPEMBlock(block, password); err != nil {
		panic(err)
	}

	if parsedKey, err = x509.ParsePKCS1PrivateKey(blockDecrypted); err != nil {
		if parsedKey, err = x509.ParsePKCS8PrivateKey(blockDecrypted); err != nil {
			panic(err)
		}
	}

	privateKey, ok := parsedKey.(*rsa.PrivateKey)
	if !ok {
		panic("key is not of type *rsa.PrivateKey")
	}

	return privateKey
}

// MustParseRSAPublicKey encodes a PEM encoded PKCS1 or PKCS8 public key.
func MustParseRSAPublicKey(key []byte) *rsa.PublicKey {
	var err error

	// Parse PEM block
	var block *pem.Block
	if block, _ = pem.Decode(key); block == nil {
		panic(ErrNotPEM)
	}

	// Parse the key
	var parsedKey interface{}
	if parsedKey, err = x509.ParsePKIXPublicKey(block.Bytes); err != nil {
		if cert, err := x509.ParseCertificate(block.Bytes); err == nil {
			parsedKey = cert.PublicKey
		} else {
			panic(err)
		}
	}

	var pkey *rsa.PublicKey
	var ok bool
	if pkey, ok = parsedKey.(*rsa.PublicKey); !ok {
		panic("key is not of type *rsa.PublicKey")
	}

	return pkey
}

/*
// MustParseEd25519 PEM encoded Ed25519.
func MustParseEd25519(key []byte) ed25519.PrivateKey {
	// Parse PEM block
	block, _ := pem.Decode(key)
	if block == nil {
		panic(ErrNotPEM)
	}

	type ed25519PrivKey struct {
		Version          int
		ObjectIdentifier struct {
			ObjectIdentifier asn1.ObjectIdentifier
		}
		PrivateKey []byte
	}

	var asn1PrivKey ed25519PrivKey
	if _, err := asn1.Unmarshal(block.Bytes, &asn1PrivKey); err != nil {
		panic(err)
	}

	privateKey := ed25519.NewKeyFromSeed(asn1PrivKey.PrivateKey[2:])
	return privateKey
}
*/
