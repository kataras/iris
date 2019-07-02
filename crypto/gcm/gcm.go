// Package gcm implements encryption/decription using the AES algorithm and the Galois/Counter Mode.
package gcm

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha512"
	"encoding/hex"
)

// MustGenerateKey generates an aes key.
// It panics if any error occurred.
func MustGenerateKey() []byte {
	aesKey, err := GenerateKey()
	if err != nil {
		panic(err)
	}

	return aesKey
}

// GenerateKey returns a random aes key.
func GenerateKey() ([]byte, error) {
	key := make([]byte, 64)
	n, err := rand.Read(key)
	if err != nil {
		return nil, err
	}

	return encode(key[:n]), nil
}

// Encrypt encrypts and authenticates the plain data and additional data
// and returns the ciphertext of it.
// It uses the AEAD cipher mode providing authenticated encryption with associated
// data.
// The same additional data must be kept the same for `Decrypt`.
func Encrypt(aesKey, data, additionalData []byte) ([]byte, error) {
	key, err := decode(aesKey)
	if err != nil {
		return nil, err
	}

	h := sha512.New()
	h.Write(key)
	digest := encode(h.Sum(nil))

	// key based on the hash itself, we have space because of sha512.
	newKey, err := decode(digest[:64])
	if err != nil {
		return nil, err
	}
	// nonce based on the hash itself.
	nonce, err := decode(digest[64:(64 + 24)])
	if err != nil {
		return nil, err
	}

	aData, err := decode(additionalData)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(newKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	ciphertext := encode(gcm.Seal(nil, nonce, data, aData))
	return ciphertext, nil
}

// Decrypt decrypts and authenticates ciphertext, authenticates the
// additional data and, if successful, returns the resulting plain data.
// The additional data must match the value passed to `Encrypt`.
func Decrypt(aesKey, ciphertext, additionalData []byte) ([]byte, error) {
	key, err := decode(aesKey)
	if err != nil {
		return nil, err
	}

	h := sha512.New()
	h.Write(key)
	digest := encode(h.Sum(nil))

	newKey, err := decode(digest[:64])
	if err != nil {
		return nil, err
	}
	nonce, err := decode(digest[64:(64 + 24)])
	if err != nil {
		return nil, err
	}

	additionalData, err = decode(additionalData)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(newKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	ciphertext, err = decode(ciphertext)
	return gcm.Open(nil, nonce, ciphertext, additionalData)
}

func decode(src []byte) ([]byte, error) {
	buf := make([]byte, hex.DecodedLen(len(src)))
	n, err := hex.Decode(buf, src)
	if err != nil {
		return nil, err
	}

	return buf[:n], nil
}

func encode(src []byte) []byte {
	buf := make([]byte, hex.EncodedLen(len(src)))
	hex.Encode(buf, src)
	return buf
}
