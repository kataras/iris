// Field validation, end to end: Validate + validation.Join -> wire format.
//
// A request type validates itself in a Validate method. ctx.BindJSON[T] calls
// it after decoding and, on a non-nil result, answers 400 INVALID_ARGUMENT
// with one entry per failing field. The builders accumulate every broken rule
// of a field into a single reason and validation.Join drops the fields that
// passed, returning nil when everything is valid.
//
// The first request under Try answers 400 with exactly this body (the id is a
// per-occurrence UUID):
//
//	{
//	  "id": "…",
//	  "http_error_code": {"canonical_name": "INVALID_ARGUMENT", "status": 400},
//	  "message": "validation failure",
//	  "detail": "fields were invalid",
//	  "validation": [
//	    {"field": "name", "value": "", "reason": "must not be empty, must be between 3 and 50 characters"},
//	    {"field": "email", "value": "bad", "reason": "must be a valid email address"},
//	    {"field": "age", "value": 9, "reason": "must be between 18 and 130"},
//	    {"field": "roles", "value": [], "reason": "must not be empty"}
//	  ]
//	}
//
// "name" shows the accumulation: both of its rules failed and their messages
// were joined into one reason.
//
// Learn: report every invalid field of a request in one 400 response, from a
// Validate method or by hand with validation.Join and errors.InvalidArgument.Err.
// Before: ../handle-errorf
//
// Run:
//
//	go run main.go
//
// Try:
//
//	curl -i -X POST http://localhost:8080/users \
//	  -d '{"name":"","email":"bad","age":9,"roles":[]}'                                  # 400 INVALID_ARGUMENT, the body above
//	curl -i -X POST http://localhost:8080/users \
//	  -d '{"name":"Jane Doe","email":"jane@example.com","age":32,"roles":["editor"]}'   # 201
//	curl -i "http://localhost:8080/users?page=0"                                        # 400, validation outside a Validate method
//
// Next: ../error-router
package main

import (
	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/rest/errors"
	"github.com/kataras/iris/v14/rest/errors/validation"
)

func main() {
	app := iris.New()

	app.Post("/users", func(ctx iris.Context) {
		input, ok := ctx.BindJSON[CreateUserInput]()
		if !ok {
			return // 400 already written: malformed JSON or invalid fields.
		}

		ctx.Created(input, nil) // 201 with the payload as JSON.
	})

	// Outside a Validate method the field errors are sent explicitly. Err
	// routes validation errors to the same wire format as above.
	app.Get("/users", func(ctx iris.Context) {
		page := validation.Number("page", ctx.Query().GetInt("page")).Positive()
		if err := validation.Join(page); err != nil {
			errors.InvalidArgument.Err(ctx, err)
			return
		}

		ctx.OK(iris.Map{"page": ctx.Query().GetInt("page"), "users": []string{}}, nil)
	})

	app.Listen(":8080")
}

// CreateUserInput is the request payload. The pointer receiver on Validate is
// what ctx.BindJSON looks for.
type CreateUserInput struct {
	Name  string   `json:"name"`
	Email string   `json:"email"`
	Age   int      `json:"age"`
	Roles []string `json:"roles"`
}

// Validate reports every invalid field at once. Each builder is chainable and
// Join returns nil when no field failed.
func (in *CreateUserInput) Validate() error {
	return validation.Join(
		validation.String("name", in.Name).NotEmpty().Length(3, 50),
		validation.String("email", in.Email).Email(),
		validation.Number("age", in.Age).InRange(18, 130),
		validation.Slice("roles", in.Roles).NotEmpty(),
	)
}
