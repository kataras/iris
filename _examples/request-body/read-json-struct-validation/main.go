// Package main shows the validator(latest, version 10) integration with Iris' Context methods of
// `ReadJSON`, `ReadXML`, `ReadMsgPack`, `ReadYAML`, `ReadForm`, `ReadQuery`, `ReadBody`.
//
// You can find more examples of this 3rd-party library at:
// https://github.com/go-playground/validator/blob/master/_examples
package main

import (
	"fmt"

	"github.com/kataras/iris/v12"

	// $ go get github.com/go-playground/validator/v10@latest
	"github.com/go-playground/validator/v10"
)

func main() {
	app := iris.New()
	app.Validator = validator.New()

	userRouter := app.Party("/user")
	{
		userRouter.Get("/validation-errors", resolveErrorsDocumentation)
		userRouter.Post("/", postUser)
	}

	// Use Postman or any tool to perform a POST request
	// to the http://localhost:8080/user with RAW BODY of:
	/*
		{
			"fname": "",
			"lname": "",
			"age": 45,
			"email": "mail@example.com",
			"favColor": "#000",
			"addresses": [{
				"street": "Eavesdown Docks",
				"planet": "Persphone",
				"phone": "none",
				"city": "Unknown"
			}]
		}
	*/
	/* The response should be:
		{
		  "title": "Validation error",
	      "detail": "One or more fields failed to be validated",
	      "type": "http://localhost:8080/user/validation-errors",
	      "status": 400,
		  "fields": [
		    {
		      "tag": "required",
		      "namespace": "User.FirstName",
		      "kind": "string",
		      "type": "string",
		      "value": "",
		      "param": ""
		    },
		    {
		      "tag": "required",
		      "namespace": "User.LastName",
		      "kind": "string",
		      "type": "string",
		      "value": "",
		      "param": ""
		    }
		  ]
		}
	*/
	app.Listen(":8080")
}

// User contains user information.
type User struct {
	FirstName      string     `json:"fname" validate:"required"`
	LastName       string     `json:"lname" validate:"required"`
	Age            uint8      `json:"age" validate:"gte=0,lte=130"`
	Email          string     `json:"email" validate:"required,email"`
	FavouriteColor string     `json:"favColor" validate:"hexcolor|rgb|rgba"`
	Addresses      []*Address `json:"addresses" validate:"required,dive,required"` // a User can have a home and cottage...
}

// Address houses a users address information.
type Address struct {
	Street string `json:"street" validate:"required"`
	City   string `json:"city" validate:"required"`
	Planet string `json:"planet" validate:"required"`
	Phone  string `json:"phone" validate:"required"`
}

type validationError struct {
	ActualTag string `json:"tag"`
	Namespace string `json:"namespace"`
	Kind      string `json:"kind"`
	Type      string `json:"type"`
	Value     string `json:"value"`
	Param     string `json:"param"`
}

func wrapValidationErrors(errs validator.ValidationErrors) []validationError {
	validationErrors := make([]validationError, 0, len(errs))
	for _, validationErr := range errs {
		validationErrors = append(validationErrors, validationError{
			ActualTag: validationErr.ActualTag(),
			Namespace: validationErr.Namespace(),
			Kind:      validationErr.Kind().String(),
			Type:      validationErr.Type().String(),
			Value:     fmt.Sprintf("%v", validationErr.Value()),
			Param:     validationErr.Param(),
		})
	}

	return validationErrors
}

func postUser(ctx iris.Context) {
	var user User
	err := ctx.ReadJSON(&user)
	if err != nil {
		// Handle the error, below you will find the right way to do that...

		if errs, ok := err.(validator.ValidationErrors); ok {
			// Wrap the errors with JSON format, the underline library returns the errors as interface.
			validationErrors := wrapValidationErrors(errs)

			// Fire an application/json+problem response and stop the handlers chain.
			ctx.StopWithProblem(iris.StatusBadRequest, iris.NewProblem().
				Title("Validation error").
				Detail("One or more fields failed to be validated").
				Type("/user/validation-errors").
				Key("errors", validationErrors))

			return
		}

		// It's probably an internal JSON error, let's dont give more info here.
		ctx.StopWithStatus(iris.StatusInternalServerError)
		return
	}

	ctx.JSON(iris.Map{"message": "OK"})
}

func resolveErrorsDocumentation(ctx iris.Context) {
	ctx.WriteString("A page that should document to web developers or users of the API on how to resolve the validation errors")
}
