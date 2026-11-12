package sdk

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"io"
)

// EncryptPassword encrypts plainPassword using AES-GCM with a shared key.
// The same password may have a different outcome, and that's the best solution so
// even the server doesn't know the actual password.
// A proper length (16, 24, or 32 bytes) for AES and is kept strictly confidential.
func EncryptPassword(plainPassword string, sharedKeyHex string) (string, error) {
	return encrypt([]byte(plainPassword), sharedKeyHex)
}

func encrypt(plaintext []byte, sharedKeyHex string) (string, error) {
	// Decode it first, as the server sends it as hex.
	sharedKey, err := hex.DecodeString(sharedKeyHex)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(sharedKey)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return hex.EncodeToString(ciphertext), nil
	// return base64.StdEncoding.EncodeToString(ciphertext), nil
}
