package jwt

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"io/ioutil"
	"os"
)

// LoadRSA tries to read RSA Private Key from "fname" system file,
// if does not exist then it generates a new random one based on "bits" (e.g. 2048, 4096)
// and exports it to a new "fname" file.
func LoadRSA(fname string, bits int) (key *rsa.PrivateKey, err error) {
	exists := fileExists(fname)
	if exists {
		key, err = importFromFile(fname)
	} else {
		key, err = rsa.GenerateKey(rand.Reader, bits)
	}

	if err != nil {
		return
	}

	if !exists {
		err = exportToFile(key, fname)
	}

	return
}

func exportToFile(key *rsa.PrivateKey, filename string) error {
	b := x509.MarshalPKCS1PrivateKey(key)
	encoded := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: b,
		},
	)

	return ioutil.WriteFile(filename, encoded, 0644)
}

func importFromFile(filename string) (*rsa.PrivateKey, error) {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	return ParseRSAPrivateKey(b, nil)
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

var (
	// ErrNotPEM is an error type of the `ParseXXX` function(s) fired
	// when the data are not PEM-encoded.
	ErrNotPEM = errors.New("key must be PEM encoded")
	// ErrInvalidKey is an error type of the `ParseXXX` function(s) fired
	// when the contents are not type of rsa private key.
	ErrInvalidKey = errors.New("key is not of type *rsa.PrivateKey")
)

// ParseRSAPrivateKey encodes a PEM-encoded PKCS1 or PKCS8 private key protected with a password.
func ParseRSAPrivateKey(key, password []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(key)
	if block == nil {
		return nil, ErrNotPEM
	}

	var (
		parsedKey interface{}
		err       error
	)

	var blockDecrypted []byte
	if len(password) > 0 {
		if blockDecrypted, err = x509.DecryptPEMBlock(block, password); err != nil {
			return nil, err
		}
	} else {
		blockDecrypted = block.Bytes
	}

	if parsedKey, err = x509.ParsePKCS1PrivateKey(blockDecrypted); err != nil {
		if parsedKey, err = x509.ParsePKCS8PrivateKey(blockDecrypted); err != nil {
			return nil, err
		}
	}

	privateKey, ok := parsedKey.(*rsa.PrivateKey)
	if !ok {
		return nil, ErrInvalidKey
	}

	return privateKey, nil
}
