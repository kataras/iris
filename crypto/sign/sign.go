// Package sign signs and verifies any format of data by
// using the ECDSA P-384 digital signature and authentication algorithm.
//
// https://en.wikipedia.org/wiki/Elliptic_Curve_Digital_Signature_Algorithm
// https://apps.nsa.gov/iaarchive/library/ia-guidance/ia-solutions-for-classified/algorithm-guidance/suite-b-implementers-guide-to-fips-186-3-ecdsa.cfm
// https://www.nsa.gov/Portals/70/documents/resources/everyone/csfc/csfc-faqs.pdf
package sign

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"  // the key encoding.
	"encoding/pem" // the data encoding format.
	"errors"
	"math/big"

	// the, modern, hash implementation,
	// commonly used in popular crypto concurrencies too.
	"golang.org/x/crypto/sha3"
)

// MustGenerateKey generates a private and public  key pair.
// It panics if any error occurred.
func MustGenerateKey() *ecdsa.PrivateKey {
	privateKey, err := GenerateKey()
	if err != nil {
		panic(err)
	}

	return privateKey
}

// GenerateKey generates a private and public key pair.
func GenerateKey() (*ecdsa.PrivateKey, error) {
	return ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
}

// GeneratePrivateKey generates a private key as pem text.
// It returns empty on any error.
func GeneratePrivateKey() string {
	privateKey, err := GenerateKey()
	if err != nil {
		return ""
	}

	privateKeyB, err := marshalPrivateKey(privateKey)
	if err != nil {
		return ""
	}

	return string(privateKeyB)
}

// Sign signs the "data" using the "privateKey".
// It returns the signature.
func Sign(privateKey *ecdsa.PrivateKey, data []byte) ([]byte, error) {
	h := sha3.New256()
	_, err := h.Write(data)
	if err != nil {
		return nil, err
	}
	digest := h.Sum(nil)

	r, s, err := ecdsa.Sign(rand.Reader, privateKey, digest)
	if err != nil {
		return nil, err
	}

	// sig := elliptic.Marshal(elliptic.P256(), r, s)
	sig := append(r.Bytes(), s.Bytes()...)

	return sig, nil
}

// Verify verifies the "data" in signature "sig" (96 length if 384) using the "publicKey".
// It reports whether the signature is valid or not.
func Verify(publicKey *ecdsa.PublicKey, sig, data []byte) (bool, error) {
	h := sha3.New256()
	_, err := h.Write(data)
	if err != nil {
		return false, err
	}

	digest := h.Sum(nil)

	// 0:32 & 32:64 for 256, always because it's constant.
	// 0:48 & 48:96 for 384 but it is not constant-time, so it's 96 or 97 length,
	// also something like that elliptic.Unmarshal(elliptic.P384(), sig)
	// doesn't work.

	r := new(big.Int).SetBytes(sig[0:32])
	s := new(big.Int).SetBytes(sig[32:64])

	return ecdsa.Verify(publicKey, digest, r, s), nil
}

var errNotValidBlock = errors.New("invalid block")

// ParsePrivateKey accepts a pem x509-encoded private key and decodes to *ecdsa.PrivateKey.
func ParsePrivateKey(key []byte) (*ecdsa.PrivateKey, error) {
	block, _ := pem.Decode(key)
	if block == nil {
		return nil, errNotValidBlock
	}
	return x509.ParseECPrivateKey(block.Bytes)
}

// ParsePublicKey accepts a pem x509-encoded public key and decodes to *ecdsa.PrivateKey.
func ParsePublicKey(key []byte) (*ecdsa.PublicKey, error) {
	block, _ := pem.Decode(key)
	if block == nil {
		return nil, errNotValidBlock
	}

	publicKeyV, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	publicKey, ok := publicKeyV.(*ecdsa.PublicKey)
	if !ok {
		return nil, errNotValidBlock
	}

	return publicKey, nil
}

func marshalPrivateKey(key *ecdsa.PrivateKey) ([]byte, error) {
	privateKeyAnsDer, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return nil, err
	}

	return pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privateKeyAnsDer}), nil
}

func marshalPublicKey(key *ecdsa.PublicKey) ([]byte, error) {
	publicKeyAnsDer, err := x509.MarshalPKIXPublicKey(key)
	if err != nil {
		return nil, err
	}

	return pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicKeyAnsDer}), nil
}
