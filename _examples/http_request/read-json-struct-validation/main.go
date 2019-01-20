// Package main shows the validator(latest, version 9) integration with Iris.
// You can find more examples like this at: https://github.com/go-playground/validator/blob/v9/_examples
package main

import (
	"fmt"

	"github.com/kataras/iris"
	// $ go get gopkg.in/go-playground/validator.v9
	"gopkg.in/go-playground/validator.v9"
)

// User contains user information.
type User struct {
	FirstName      string     `json:"fname"`
	LastName       string     `json:"lname"`
	Age            uint8      `json:"age" validate:"gte=0,lte=130"`
	Email          string     `json:"email" validate:"required,email"`
	FavouriteColor string     `json:"favColor" validate:"hexcolor|rgb|rgba"`
	Addresses      []*Address `json:"addresses" validate:"required,dive,required"` // a person can have a home and cottage...
}

// Address houses a users address information.
type Address struct {
	Street string `json:"street" validate:"required"`
	City   string `json:"city" validate:"required"`
	Planet string `json:"planet" validate:"required"`
	Phone  string `json:"phone" validate:"required"`
}

// Use a single instance of Validate, it caches struct info.
var validate *validator.Validate

func main() {
	validate = validator.New()

	// Register validation for 'User'
	// NOTE: only have to register a non-pointer type for 'User', validator
	// internally dereferences during it's type checks.
	validate.RegisterStructValidation(UserStructLevelValidation, User{})

	app := iris.New()
	app.Post("/user", func(ctx iris.Context) {
		var user User
		if err := ctx.ReadJSON(&user); err != nil {
			// Handle error.
		}

		// Returns InvalidValidationError for bad validation input, nil or ValidationErrors ( []FieldError )
		err := validate.Struct(user)
		if err != nil {

			// This check is only needed when your code could produce
			// an invalid value for validation such as interface with nil
			// value most including myself do not usually have code like this.
			if _, ok := err.(*validator.InvalidValidationError); ok {
				ctx.StatusCode(iris.StatusInternalServerError)
				ctx.WriteString(err.Error())
				return
			}

			ctx.StatusCode(iris.StatusBadRequest)
			for _, err := range err.(validator.ValidationErrors) {
				fmt.Println()
				fmt.Println(err.Namespace())
				fmt.Println(err.Field())
				fmt.Println(err.StructNamespace()) // Can differ when a custom TagNameFunc is registered or.
				fmt.Println(err.StructField())     // By passing alt name to ReportError like below.
				fmt.Println(err.Tag())
				fmt.Println(err.ActualTag())
				fmt.Println(err.Kind())
				fmt.Println(err.Type())
				fmt.Println(err.Value())
				fmt.Println(err.Param())
				fmt.Println()

				// Or collect these as json objects
				// and send back to the client the collected errors via ctx.JSON
				// {
				// 	"namespace":        err.Namespace(),
				// 	"field":            err.Field(),
				// 	"struct_namespace": err.StructNamespace(),
				// 	"struct_field":     err.StructField(),
				// 	"tag":              err.Tag(),
				// 	"actual_tag":       err.ActualTag(),
				// 	"kind":             err.Kind().String(),
				// 	"type":             err.Type().String(),
				// 	"value":            fmt.Sprintf("%v", err.Value()),
				// 	"param":            err.Param(),
				// }
			}

			// from here you can create your own error messages in whatever language you wish.
			return
		}

		// save user to database.
	})

	// use Postman or whatever to do a POST request
	// to the http://localhost:8080/user with RAW BODY:
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
	// Content-Type to application/json (optionally but good practise).
	// This request will fail due to the empty `User.FirstName` (fname in json)
	// and `User.LastName` (lname in json).
	// Check your iris' application terminal output.
	app.Run(iris.Addr(":8080"), iris.WithoutServerError(iris.ErrServerClosed))
}

// UserStructLevelValidation contains custom struct level validations that don't always
// make sense at the field validation level. For Example this function validates that either
// FirstName or LastName exist; could have done that with a custom field validation but then
// would have had to add it to both fields duplicating the logic + overhead, this way it's
// only validated once.
//
// NOTE: you may ask why wouldn't I just do this outside of validator, because doing this way
// hooks right into validator and you can combine with validation tags and still have a
// common error output format.
func UserStructLevelValidation(sl validator.StructLevel) {

	user := sl.Current().Interface().(User)

	if len(user.FirstName) == 0 && len(user.LastName) == 0 {
		sl.ReportError(user.FirstName, "FirstName", "fname", "fnameorlname", "")
		sl.ReportError(user.LastName, "LastName", "lname", "fnameorlname", "")
	}

	// plus can to more, even with different tag than "fnameorlname".
}
