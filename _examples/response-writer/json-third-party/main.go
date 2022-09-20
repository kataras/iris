package main

import (
	"github.com/kataras/iris/v12"

	// This is a 3rd-party library, which you can use to override the default behavior of ctx.JSON method.
	"github.com/bytedance/sonic"
)

func init() {
	applyIrisGlobalPatches() // <- IMPORTANT.
}

func applyIrisGlobalPatches() {
	var json = sonic.ConfigFastest

	// Apply global modifications to the context REST writers
	// without modifications to your web server's handlers code.
	iris.Patches().Context().Writers().JSON(func(ctx iris.Context, v interface{}, options *iris.JSON) error {
		enc := json.NewEncoder(ctx.ResponseWriter())
		enc.SetEscapeHTML(!options.UnescapeHTML)
		enc.SetIndent("", options.Indent)
		return enc.Encode(v)
	})
}

// User example struct for json.
type User struct {
	Firstname string `json:"firstname"`
	Lastname  string `json:"lastname"`
	City      string `json:"city"`
	Age       int    `json:"age"`
}

func main() {
	app := iris.New()

	app.Get("/", func(ctx iris.Context) {
		user := User{
			Firstname: "Gerasimos",
			Lastname:  "Maropoulos",
			City:      "Athens",
			Age:       29,
		}

		// Use ctx.JSON as you used to.
		ctx.JSON(user)
	})

	app.Listen(":8080")
}
