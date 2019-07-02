package sign

import (
	"reflect"
	"testing"
)

var (
	testPrivateKey = MustGenerateKey()
	testPublicKey  = &testPrivateKey.PublicKey
)

func TestGenerateKey(t *testing.T) {
	privateKeyB, err := marshalPrivateKey(testPrivateKey)
	if err != nil {
		t.Fatalf("private key: %v", err)
	}
	publicKeyB, err := marshalPublicKey(testPublicKey)
	if err != nil {
		t.Fatalf("public key: %v", err)
	}

	t.Logf("%s", privateKeyB)
	t.Logf("%s", publicKeyB)

	privateKeyParsed, err := ParsePrivateKey(privateKeyB)
	if err != nil {
		t.Fatalf("private key: %v", err)
	}

	publicKeyParsed, err := ParsePublicKey(publicKeyB)
	if err != nil {
		t.Fatalf("public key: %v", err)
	}

	if !reflect.DeepEqual(testPrivateKey, privateKeyParsed) {
		t.Fatalf("expected private key to be:\n%#+v\nbut got:\n%#+v", testPrivateKey, privateKeyParsed)
	}
	if !reflect.DeepEqual(testPublicKey, publicKeyParsed) {
		t.Fatalf("expected public key to be:\n%#+v\nbut got:\n%#+v", testPublicKey, publicKeyParsed)
	}
}

func TestSignAndVerify(t *testing.T) {
	tests := []struct {
		payload []byte
	}{
		{[]byte("test my content 1")},
		{[]byte("test my content 2")},
	}

	for i, tt := range tests {
		sig, err := Sign(testPrivateKey, tt.payload)
		if err != nil {
			t.Fatalf("[%d] sign error: %v", i, err)
		}

		ok, err := Verify(testPublicKey, sig, tt.payload)
		if err != nil {
			t.Fatalf("[%d] verify error: %v", i, err)
		}
		if !ok {
			t.Fatalf("[%d] verification failed for '%s'", i, tt.payload)
		}

		// test with other, invalid public key, should fail to verify.
		tempPublicKey := &MustGenerateKey().PublicKey

		ok, err = Verify(tempPublicKey, sig, tt.payload)
		if ok {
			t.Fatalf("[%d] verification should fail but passed for '%s'", i, tt.payload)
		}
	}
}
