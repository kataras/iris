package crypto

import (
	"crypto/ecdsa"
	"encoding/base64"

	"github.com/kataras/iris/crypto/gcm"
	"github.com/kataras/iris/crypto/sign"
)

var (
	// MustGenerateKey generates an ecdsa private and public key pair.
	// It panics if any error occurred.
	MustGenerateKey = sign.MustGenerateKey
	// ParsePrivateKey accepts a pem x509-encoded private key and decodes to *ecdsa.PrivateKey.
	ParsePrivateKey = sign.ParsePrivateKey
	// ParsePublicKey accepts a pem x509-encoded public key and decodes to *ecdsa.PrivateKey.
	ParsePublicKey = sign.ParsePublicKey

	// MustGenerateAESKey generates an aes key.
	// It panics if any error occurred.
	MustGenerateAESKey = gcm.MustGenerateKey
	// DefaultADATA is the default associated data used for `Encrypt` and `Decrypt`
	// when "additionalData" is empty.
	DefaultADATA = []byte("FFA0A43EA6B8C829AD403817B2F5B7A2")
)

// Encryption is the method signature when data should be signed and returned as encrypted.
type Encryption func(privateKey *ecdsa.PrivateKey, data []byte) ([]byte, error)

// Decryption is the method signature when data should be decrypted before signed.
type Decryption func(publicKey *ecdsa.PublicKey, data []byte) ([]byte, error)

// Encrypt returns an `Encryption` option to be used on `Marshal`.
// If "aesKey" is not empty then the "data" associated with the "additionalData" will be encrypted too.
// If "aesKey" is not empty but "additionalData" is, then the `DefaultADATA` will be used to encrypt "data".
// If "aesKey" is empty then encryption is disabled, the return value will be only signed.
//
// See `Unmarshal` and `Decrypt` too.
func Encrypt(aesKey, additionalData []byte) Encryption {
	if len(aesKey) == 0 {
		return nil
	}

	if len(additionalData) == 0 {
		additionalData = DefaultADATA
	}

	return func(_ *ecdsa.PrivateKey, plaintext []byte) ([]byte, error) {
		return gcm.Encrypt(aesKey, plaintext, additionalData)
	}
}

// Decrypt returns an `Decryption` option to be used on `Unmarshal`.
// If "aesKey" is not empty then the result will be decrypted.
// If "aesKey" is not empty but "additionalData" is,
// then the `DefaultADATA` will be used to decrypt the encrypted "data".
// If "aesKey" is empty then decryption is disabled.
//
// If `Marshal` had an `Encryption` then `Unmarshal` must have also.
//
// See `Marshal` and `Encrypt` too.
func Decrypt(aesKey, additionalData []byte) Decryption {
	if len(aesKey) == 0 {
		return nil
	}

	if len(additionalData) == 0 {
		additionalData = DefaultADATA
	}

	return func(_ *ecdsa.PublicKey, ciphertext []byte) ([]byte, error) {
		return gcm.Decrypt(aesKey, ciphertext, additionalData)
	}
}

// Marshal signs and, optionally, encrypts the "data".
//
// The form of the output value is: signature_of_88_length followed by the raw_data_or_encrypted_data,
// i.e "R+eqxA3LslRif0KoxpevpNILAs4Kh4mccCCoE0sRjICkj9xy0/gsxeUd2wfcGK5mzIZ6tM3A939Wjif0xwZCog==7001f30..."
//
//
// Returns non-nil error if any error occurred.
//
// Usage:
// data, _ := ioutil.ReadAll(ctx.Request().Body)
// signedData, err := crypto.Marshal(testPrivateKey, data, nil)
// ctx.Write(signedData)
// Or if data should be encrypted:
// signedEncryptedData, err := crypto.Marshal(testPrivateKey, data, crypto.Encrypt(aesKey, nil))
func Marshal(privateKey *ecdsa.PrivateKey, data []byte, encrypt Encryption) ([]byte, error) {
	if encrypt != nil {
		b, err := encrypt(privateKey, data)
		if err != nil {
			return nil, err
		}

		data = b
	}

	// sign the encrypted data if "encrypt" exists.
	sig, err := sign.Sign(privateKey, data)
	if err != nil {
		return nil, err
	}

	buf := make([]byte, base64.StdEncoding.EncodedLen(len(sig)))
	base64.StdEncoding.Encode(buf, sig)

	return append(buf, data...), nil
}

// Unmarshal verifies the "data" and, optionally, decrypts the output.
//
// Returns returns the signed raw data; without the signature and decrypted if "decrypt" is not nil.
// The second output value reports whether the verification and any decryption of the data succeed or not.
//
// Usage:
// data, _ := ioutil.ReadAll(ctx.Request().Body)
// verifiedPlainPayload, err := crypto.Unmarshal(ecdsaPublicKey, data, nil)
// ctx.Write(verifiedPlainPayload)
// Or if data are encrypted and they should be decrypted:
// verifiedDecryptedPayload, err := crypto.Unmarshal(ecdsaPublicKey, data, crypto.Decrypt(aesKey, nil))
func Unmarshal(publicKey *ecdsa.PublicKey, data []byte, decrypt Decryption) ([]byte, bool) {
	if len(data) <= 90 {
		return nil, false
	}

	sig, body := data[0:88], data[88:]

	buf := make([]byte, base64.StdEncoding.DecodedLen(len(sig)))
	n, err := base64.StdEncoding.Decode(buf, sig)
	if err != nil {
		return nil, false
	}
	sig = buf[:n]

	// verify the encrypted data as they are, the signature is linked with these.
	ok, err := sign.Verify(publicKey, sig, body)
	if !ok || err != nil {
		return nil, false
	}

	// try to decrypt the body and finally return it as plain, its original form.
	if decrypt != nil {
		body, err = decrypt(publicKey, body)
		if err != nil {
			return nil, false
		}
	}

	return body, ok && err == nil
}
