package gcm

import (
	"bytes"
	"testing"
)

var testKey = MustGenerateKey()

func TestEncryptDecrypt(t *testing.T) {
	if len(testKey) == 0 {
		t.Fatalf("testKey is empty??")
	}

	tests := []struct {
		payload []byte
		aData   []byte // IV of a random aes-256-cbc, 32 size.
	}{
		{[]byte("test my content 1"), []byte("FFA0A43EA6B8C829AD403817B2F5B7A2")},
		{[]byte("test my content 2"), []byte("364787B9AF1AEE4BE26690EA8CBF4AB7")},
	}

	for i, tt := range tests {
		ciphertext, err := Encrypt(testKey, tt.payload, tt.aData)
		if err != nil {
			t.Fatalf("[%d] encrypt error: %v", i, err)
		}

		payload, err := Decrypt(testKey, ciphertext, tt.aData)
		if err != nil {
			t.Fatalf("[%d] decrypt error: %v", i, err)
		}

		if !bytes.Equal(payload, tt.payload) {
			t.Fatalf("[%d] expected data to be decrypted to: '%s' but got: '%s'", i, tt.payload, payload)
		}

		// test with other, invalid key, should fail to decrypt.
		tempKey := MustGenerateKey()

		payload, err = Decrypt(tempKey, ciphertext, tt.aData)
		if err == nil || len(payload) > 0 {
			t.Fatalf("[%d] verification should fail but passed for '%s'", i, tt.payload)
		}
	}
}
