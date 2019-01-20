package main

import (
	"github.com/kataras/iris"
)

// NOTE: THIS IS OPTIONALLY.
// It is just an example of communication between cors/simple/main.go and your app
// based on issues that beginners had with it.
// You should use your own favourite library for HTTP requests (any programming language ofc).
//
// Replace the '8fc93b1c.ngrok.io' with a domain which
// exposes the cors/simple/main.go server side.
const url = "http://8fc93b1c.ngrok.io/api/v1/mailer"

var clientSide = []byte(`<script type="text/javascript">
fetch("` + url + `", {
	headers: {
	  "Content-Type": "application/json",
	  "Access-Control-Allow-Origin": "*"
	},
	method: "POST",
	mode: "cors",
	body: JSON.stringify({ email: "mymail@mail.com" }),
});
</script>`)

func main() {
	app := iris.New()
	app.Get("/", func(ctx iris.Context) {
		ctx.Write(clientSide)
	})

	// Start and navigate to http://localhost:8080
	// and go to the previous terminal of your running cors/simple/main.go server
	// and see the logs.
	app.Run(iris.Addr(":8080"))
}
