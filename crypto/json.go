package crypto

import (
	"crypto/ecdsa"
	"encoding/base64"
	"encoding/json"
	"io"
	"io/ioutil"

	"github.com/kataras/iris/crypto/sign"
)

// Ticket contains the original payload raw data
// and the generated signature.
//
// Look `SignJSON` and `VerifyJSON` for more details.
type Ticket struct {
	Payload   json.RawMessage `json:"payload"`
	Signature string          `json:"signature"`
}

// SignJSON signs the incoming JSON request payload based on
// client's "privateKey" and the "r" (could be ctx.Request().Body).
//
// It generates the signature and returns a structure called `Ticket`.
// The `Ticket` just contains the original client's payload raw data
// and the generated signature.
//
// Returns non-nil error if any error occurred.
//
// Usage:
// ticket, err := crypto.SignJSON(testPrivateKey, ctx.Request().Body)
// b, err := json.Marshal(ticket)
// ctx.Write(b)
func SignJSON(privateKey *ecdsa.PrivateKey, r io.Reader) (Ticket, error) {
	data, err := ioutil.ReadAll(r)
	if err != nil || len(data) == 0 {
		return Ticket{}, err
	}

	sig, err := sign.Sign(privateKey, data)
	if err != nil {
		return Ticket{}, err
	}

	ticket := Ticket{
		Payload:   data,
		Signature: base64.StdEncoding.EncodeToString(sig),
	}
	return ticket, nil
}

// VerifyJSON verifies the incoming JSON request,
// by reading the "r" which should decodes to a `Ticket`.
// The `Ticket` is verified against the given "publicKey", the `Ticket#Signature` and
// `Ticket#Payload` data (original request's payload data which was signed by `SignJSON`).
//
// Returns true whether the verification succeed or not.
// The "toPayloadPtr" should be a pointer to a value of the same payload structure the client signed on.
// If and only if the verification succeed the payload value is filled from the `Ticket.Payload` raw data.
//
// Check for both output arguments in order to:
// 1. verification (true/false and error) and
// 2. ticket's original json payload parsed and "toPayloadPtr" is filled successfully (error).
//
// Usage:
// var myPayload myJSONStruct
// ok, err := crypto.VerifyJSON(publicKey, ctx.Request().Body, &myPayload)
func VerifyJSON(publicKey *ecdsa.PublicKey, r io.Reader, toPayloadPtr interface{}) (bool, error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return false, err
	}

	ticket := new(Ticket)
	err = json.Unmarshal(data, ticket)
	if err != nil {
		return false, err
	}

	sig, err := base64.StdEncoding.DecodeString(ticket.Signature)
	if err != nil {
		return false, err
	}

	ok, err := sign.Verify(publicKey, sig, ticket.Payload)
	if ok && toPayloadPtr != nil {
		// if and only if the verification succeed we
		// set the payload to the structured/map value of "toPayloadPtr".
		err = json.Unmarshal(ticket.Payload, toPayloadPtr)
	}

	return ok, err
}
