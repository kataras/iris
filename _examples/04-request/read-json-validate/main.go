// Reading and validating a JSON body with ctx.BindJSON[T].
//
// A payload type that implements Validate() error (the iris.Validatable
// interface) is validated automatically after decoding: ctx.BindJSON writes
// the 400 response itself and returns false, so the handler only ever sees
// decoded, valid input. The rest/errors/validation builders (String, Number,
// Join) produce one entry per failed field; Join drops the fields that passed
// and returns nil when everything did.
//
// Learn: decode and validate a JSON body in one call and know the exact 400
// clients receive.
// Before: ../query-and-forms
//
// Run:
//
//	go run main.go
//
// Try:
//
//	curl -i -X POST http://localhost:8080/users -H 'Content-Type: application/json' \
//	  -d '{"name":"Gerasimos Maropoulos","email":"kataras2006@hotmail.com","age":36}'   # 201, the payload echoed
//	curl -i -X POST http://localhost:8080/users -H 'Content-Type: application/json' \
//	  -d '{"name":"","email":"nope","age":0}'                                            # 400, one entry per field
//
// The 400 body, where "id" is a per-error tracking id:
//
//	{
//	  "id": "2dd92ea9-1058-4874-b096-c80269a826a0",
//	  "http_error_code": {"canonical_name":"INVALID_ARGUMENT","status":400},
//	  "message": "validation failure",
//	  "detail":  "fields were invalid",
//	  "validation": [
//	    {"field":"name","value":"","reason":"must not be empty, must be between 2 and 100 characters"},
//	    {"field":"email","value":"nope","reason":"must be a valid email address"},
//	    {"field":"age","value":0,"reason":"must be positive"}
//	  ]
//	}
//
// Next: ../read-json-optional
package main

import (
	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/rest/errors/validation"
)

func main() {
	app := iris.New()
	app.Post("/users", createUser)
	app.Listen(":8080")
}

// CreateUserInput is the request payload.
type CreateUserInput struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Age   int    `json:"age"`
}

// Validate implements iris.Validatable: ctx.BindJSON calls it after a
// successful decode, so no handler ever calls it by hand.
func (in CreateUserInput) Validate() error {
	return validation.Join(
		validation.String("name", in.Name).NotEmpty().Length(2, 100),
		validation.String("email", in.Email).Email(),
		validation.Number("age", in.Age).Positive(),
	)
}

func createUser(ctx iris.Context) {
	input, ok := ctx.BindJSON[CreateUserInput]()
	if !ok {
		return // 400 already written: malformed JSON or a failed Validate.
	}

	// input is decoded AND valid from here on.
	ctx.Created(input, nil) // 201 with the payload as JSON.
}
