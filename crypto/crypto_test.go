package crypto

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

var (
	testPrivateKey = MustGenerateKey()
	testPublicKey  = &testPrivateKey.PublicKey
	testAESKey     = MustGenerateAESKey()
)

func TestMarshalAndUnmarshal(t *testing.T) {
	testPayloadData := []byte(`{"mykey":"myvalue","mysecondkey@":"mysecondv#lu3@+!"}!+,==||any<data>[here]`)

	signHandler := func(w http.ResponseWriter, r *http.Request) {
		data, _ := ioutil.ReadAll(r.Body)
		signedEncryptedPayload, err := Marshal(testPrivateKey, data, Encrypt(testAESKey, nil))
		if err != nil {
			w.WriteHeader(http.StatusUnprocessableEntity)
			return
		}

		w.Write(signedEncryptedPayload)
	}

	verifyHandler := func(w http.ResponseWriter, r *http.Request) {
		publicKey := testPublicKey
		if r.URL.Path == "/verify/otherkey" {
			// test with other, generated, public key.
			publicKey = &MustGenerateKey().PublicKey
		}
		data, _ := ioutil.ReadAll(r.Body)
		payload, ok := Unmarshal(publicKey, data, Decrypt(testAESKey, nil))
		if !ok {
			w.WriteHeader(http.StatusUnprocessableEntity)
			return
		}

		//	re-send the payload.
		w.Write(payload)
	}

	testPayload := testPayloadData
	t.Logf("signing: sending payload: %s", testPayload)

	signRequest := httptest.NewRequest("POST", "/sign", bytes.NewBuffer(testPayload))
	signRec := httptest.NewRecorder()
	signHandler(signRec, signRequest)

	gotSignedEncrypted, _ := ioutil.ReadAll(signRec.Body)

	// Looks like this:
	// jWQIL5gqTd1JqyHoTDXSaEtOmJdpYuzU0cyEn/9uDMW2JcPi4FkYfkkCfKyLFzlwhbykXsSJXOV11yVnS3EG4w==885c46964d92cce1fb36f9dfd76f2003000338e8605cd59fd0b5a84abf8175c41bf8bdbac0327cbc3cec17bf42ff9c
	t.Logf("verification: sending signed encrypted payload:\n%s", gotSignedEncrypted)
	verifyRequest := httptest.NewRequest("POST", "/verify", bytes.NewBuffer(gotSignedEncrypted))
	verifyRec := httptest.NewRecorder()
	verifyHandler(verifyRec, verifyRequest)
	verifyRequest.Body.Close()

	if expected, got := http.StatusOK, verifyRec.Code; expected != got {
		t.Fatalf("verification: expected status code: %d but got: %d", expected, got)
	}

	gotPayload, err := ioutil.ReadAll(verifyRec.Body)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(testPayload, gotPayload) {
		t.Fatalf("verification: expected payload: '%s' but got: '%s'", testPayload, gotPayload)
	}

	t.Logf("got plain payload:\n%s\n\n", gotPayload)

	// test the same payload, with the same signature but with other public key (see handler checks the path for that).
	t.Logf("verification: sending the same signed encrypted data which should not be verified due to a different key pair...")
	verifyRequest = httptest.NewRequest("POST", "/verify/otherkey", bytes.NewBuffer(gotSignedEncrypted))
	verifyRec = httptest.NewRecorder()
	verifyHandler(verifyRec, verifyRequest)
	verifyRequest.Body.Close()

	if expected, got := http.StatusUnprocessableEntity, verifyRec.Code; expected != got {
		t.Fatalf("verification: expected status code: %d but got: %d", expected, got)
	}

	gotPayload, _ = ioutil.ReadAll(verifyRec.Body)
	if len(gotPayload) > 0 {
		t.Fatalf("verification should fail and no payload should return but got: '%s'", gotPayload)
	}

	t.Logf("correct, it didn't match")
}
