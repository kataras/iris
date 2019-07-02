package main

import (
	"io/ioutil"

	"github.com/kataras/iris"
	"github.com/kataras/iris/crypto"
)

var (
	// Change that to your owns, usally you have an ECDSA private key
	// per identify, let's say a user, stored in a database
	// or somewhere else and you use its public key
	// to sign a user's payload and when this client
	// wants to use this payload, on another route,
	// you verify it comparing the signature of the payload
	// with the user's public key.
	//
	// Use the crypto.MustGenerateKey to generate a random key
	// or import
	// the "github.com/kataras/iris/crypto/sign"
	// and use its
	// sign.ParsePrivateKey/ParsePublicKey(theKey []byte)
	// to convert data or local file to an *ecdsa.PrivateKey.
	testPrivateKey = crypto.MustGenerateKey()
	testPublicKey  = &testPrivateKey.PublicKey
)

type testPayloadStructure struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// The Iris crypto package offers
// authentication (with optional encryption in top of) and verification
// of raw []byte data with `crypto.Marshal/Unmarshal` functions
// and JSON payloads with `crypto.SignJSON/VerifyJSON functions.
//
// Let's use the `SignJSON` and `VerifyJSON` here as an example,
// as this is the most common scenario for a web application.
func main() {
	app := iris.New()

	app.Post("/auth/json", func(ctx iris.Context) {
		ticket, err := crypto.SignJSON(testPrivateKey, ctx.Request().Body)
		if err != nil {
			ctx.StatusCode(iris.StatusUnprocessableEntity)
			return
		}

		// Send just the signature back
		// ctx.WriteString(ticket.Signature)
		// or the whole payload + the signature:
		ctx.JSON(ticket)
	})

	app.Post("/verify/json", func(ctx iris.Context) {
		var verificatedPayload testPayloadStructure // this can be anything.

		// The VerifyJSON excepts the body to be a JSON structure of
		// {
		//   "signature": the generated signature from /auth/json,
		//   "payload": the JSON client payload
		// }
		// That is the form of the `crypto.Ticket` structure.
		//
		// However, you are not limited to use that form, another common practise is to
		// have the signature and the payload we need to check in the same string representation
		// and for a better security you add encryption in top of it, so an outsider cannot understand what is what.
		// Let's say that the signature can be optionally provided by a URL ENCODED parameter
		// and the request body is the payload without any encryption
		// -
		// of course you can pass an GCM type of encryption/decryption as Marshal's and Unmarshal's last input argument,
		// see more about this at the iris/crypto/gcm subpackage for ready-to-use solutions.
		// -
		// So we will check if a url parameter is given, if so we will combine the signature and the body into one slice of bytes
		// and we will make use of the `crypto.Unmarshal` instead of the `crypto.VerifyJSON` function
		// -
		if signature := ctx.URLParam("signature"); signature != "" {
			payload, err := ioutil.ReadAll(ctx.Request().Body)
			if err != nil {
				ctx.StatusCode(iris.StatusInternalServerError)
				return
			}

			data := append([]byte(signature), payload...)

			originalPayloadBytes, ok := crypto.Unmarshal(testPublicKey, data, nil)

			if !ok {
				ctx.Writef("this does not match, please try again\n")
				ctx.StatusCode(iris.StatusUnprocessableEntity)
				return
			}

			ctx.ContentType("application/json")
			ctx.Write(originalPayloadBytes)
			return
		}

		ok, err := crypto.VerifyJSON(testPublicKey, ctx.Request().Body, &verificatedPayload)
		if err != nil {
			ctx.Writef("error on verification: %v\n", err)
			ctx.StatusCode(iris.StatusBadRequest)
			return
		}

		if !ok {
			ctx.Writef("this does not match, please try again\n")
			ctx.StatusCode(iris.StatusUnprocessableEntity)
			return
		}

		// Give back the verificated payload or use it.
		ctx.JSON(verificatedPayload)
	})

	// 1.
	// curl -X POST -H "Content-Type: application/json" -d '{"key": "this is a key", "value": "this is a value"}' http://localhost:8080/auth/json
	// 2. The result will be something like this:
	// {"payload":{"key":"this is a key","value":"this is a value"},"signature":"UgXgbXXvs9nAB3Pg0mG1WR0KBn2KpD/xBIsyOv1o4ZpzKs45hB/yxXiGN1k4Y+mgjdBxP6Gg26qajK6216pAGA=="}
	// 3. Copy-paste the whole result and do:
	// curl -X POST -H "Content-Type: application/json" -d '{"payload":{"key":"this is a key","value":"this is a value"},"signature":"UgXgbXXvs9nAB3Pg0mG1WR0KBn2KpD/xBIsyOv1o4ZpzKs45hB/yxXiGN1k4Y+mgjdBxP6Gg26qajK6216pAGA=="}' http://localhost:8080/verify/json
	// 4. Or pass by ?signature encoded URL parameter:
	// curl -X POST -H "Content-Type: application/json" -d '{"key": "this is a key", "value": "this is a value"}' http://localhost:8080/verify/json?signature=UgXgbXXvs9nAB3Pg0mG1WR0KBn2KpD%2FxBIsyOv1o4ZpzKs45hB%2FyxXiGN1k4Y%2BmgjdBxP6Gg26qajK6216pAGA%3D%3D
	// 5. At both cases the result should be:
	// {"key":"this is a key","value":"this is a value"}
	// Otherise the verification failed.
	//
	// Note that each time server is restarted a new private and public key pair is generated,
	// look at the start of the program.
	app.Run(iris.Addr(":8080"))
}

// You can read more examples and run testable code
// at the `iris/crypto`, `iris/crypto/sign`
// and `iris/crypto/gcm` packages themselves.
