package identity

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"image/png"
	"time"

	"github.com/kataras/jwt"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

// GenerateOTP generates a new TOTP key and returns the QR code image and the key.
func GenerateOTP(issuer, userID, username string) (key *otp.Key, img []byte, err error) {
	key, err = totp.Generate(totp.GenerateOpts{
		Issuer:      issuer,
		AccountName: username,
		Period:      30,
		SecretSize:  15,
		Digits:      otp.DigitsSix,
	})
	if err != nil {
		return
	}

	// Convert TOTP key into a QR code encoded as a PNG image.
	var buf bytes.Buffer
	imgData, err := key.Image(200, 200)
	if err != nil {
		return nil, nil, err
	}
	png.Encode(&buf, imgData)
	img = buf.Bytes()

	return
}

// ValidateOTP validates the given passcode against the given OTP AUTH URL.
func ValidateOTP(passcode, otpURL string) (bool, error) {
	key, err := otp.NewKeyFromURL(otpURL)
	if err != nil {
		return false, err
	}

	valid := totp.Validate(passcode, key.Secret())
	return valid, nil
}

// GenerateOTPCode generates a new OTP code based on the given OTP AUTH URL.
func GenerateOTPCode(otpURL string) (string, error) {
	key, err := otp.NewKeyFromURL(otpURL)
	if err != nil {
		return "", err
	}

	// totp code is just an HOTP code seeded with every 30 seconds.
	interval := time.Now().Unix() / 30
	return createCode(key.Secret(), interval)
}

// createCode creates a new OTP code based on either a time or counter interval.
// The time is used for TOTP and the counter is used for HOTP algorithm.
func createCode(secret string, interval int64) (string, error) {
	sec, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(secret)
	if err != nil {
		return "", err
	}

	hash := hmac.New(sha1.New, sec)
	if err := binary.Write(hash, binary.BigEndian, interval); err != nil {
		return "", err
	}
	sign := hash.Sum(nil)

	offset := sign[19] & 15
	trunc := binary.BigEndian.Uint32(sign[offset : offset+4])

	return fmt.Sprintf("%0*d", otp.DigitsSix, (trunc&0x7fffffff)%1000000), nil
}

const (
	// RecoveryCodesLength is the array length of the recovery codes.
	RecoveryCodesLength = 4
	// RecoveryCodeLength is the character length of each recovery code.
	RecoveryCodeLength = 12
)

// GenerateRecoveryCodes generates 4 recovery codes of 12 chars length that can be used only once.
func GenerateRecoveryCodes() []string {
	// Generate RecoveryCodesLength recovery codes
	recoveryCodes := make([]string, RecoveryCodesLength)
	for i := range recoveryCodes {
		recoveryCodes[i] = generateUniqueRecoveryCode(recoveryCodes) // make sure it's unique.
	}

	return recoveryCodes
}

func generateUniqueRecoveryCode(existingCodes []string) string {
	code := jwt.MustGenerateRandomString(RecoveryCodeLength)

	for _, existingCode := range existingCodes {
		if code == existingCode {
			code = generateUniqueRecoveryCode(existingCodes)
		}
	}

	return code
}
