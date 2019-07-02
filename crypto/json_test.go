package crypto

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestJSONSignAndVerify(t *testing.T) {
	type testJSON struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}

	signHandler := func(w http.ResponseWriter, r *http.Request) {
		ticket, err := SignJSON(testPrivateKey, r.Body)
		if err != nil {
			// https://developer.mozilla.org/en-US/docs/Web/HTTP/Status/422
			w.WriteHeader(http.StatusUnprocessableEntity)
			return
		}

		b, err := json.Marshal(ticket)
		if err != nil {
			t.Fatal(err)
		}
		w.Write(b)
		// or
		// fmt.Fprintf(w, "%s", ticket.Signature)
		// to send just the signature.
	}

	verifyHandler := func(w http.ResponseWriter, r *http.Request) {
		publicKey := testPublicKey
		if r.URL.Path == "/verify/otherkey" {
			// test with other, generated, public key.
			publicKey = &MustGenerateKey().PublicKey
		}

		var payload testJSON
		ok, err := VerifyJSON(publicKey, r.Body, &payload)
		if err != nil {
			t.Fatal(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if !ok {
			w.WriteHeader(http.StatusUnprocessableEntity) // or forbidden or unauthorized.
			return
		}

		// re-send the payload.
		b, err := json.Marshal(payload)
		if err != nil {
			t.Fatal(err)
		}
		w.Write(b)
	}

	// Looks like this:
	// {"key":"mykey","value":"myvalue"}
	testPayload := testJSON{"mykey", "myvalue"}
	payload, _ := json.Marshal(testPayload)
	t.Logf("signing: sending payload: %s", payload)

	signRequest := httptest.NewRequest("POST", "/sign", bytes.NewBuffer(payload))
	signRec := httptest.NewRecorder()
	signHandler(signRec, signRequest)

	gotTicketPayload, _ := ioutil.ReadAll(signRec.Body)

	// Looks like this:
	// {
	//	"signature": "D4PF6Hc0CrsO6MXAPxsLdhrVLKdmUOsN3Qm/Dr1y8yS80FQSgpU8Frr81fAJSKNwwW3dHhpoYvRi0t04MrukOQ==",
	//	"payload": {"key":"mykey","value":"myvalue"}
	// }
	t.Logf("verification: sending ticket: %s", gotTicketPayload)
	verifyRequest := httptest.NewRequest("POST", "/verify", bytes.NewBuffer(gotTicketPayload))
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
	if !bytes.Equal(payload, gotPayload) {
		t.Fatalf("verification: expected payload: '%s' but got: '%s'", payload, gotTicketPayload)
	}

	// test the same payload, with the same signature but with other public key (see handler checks the path for that).
	t.Logf("verification: sending the same ticket which should not be verified due to a different key pair...")
	verifyRequest = httptest.NewRequest("POST", "/verify/otherkey", bytes.NewBuffer(gotTicketPayload))
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
}
