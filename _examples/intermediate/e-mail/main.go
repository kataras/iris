package main

import (
	"bytes"

	"github.com/kataras/go-mailer"
	"gopkg.in/kataras/iris.v6"
	"gopkg.in/kataras/iris.v6/adaptors/view"
	"gopkg.in/kataras/iris.v6/adaptors/httprouter"
)

func main() {

	app := iris.New()
	// output startup banner and error logs on os.Stdout
	app.Adapt(iris.DevLogger())
	// set the router, you can choose gorillamux too
	app.Adapt(httprouter.New())
	// set a root for our templates
	app.Adapt(view.HTML("./templates", ".html"))

	// change these to your own settings
	cfg := mailer.Config{
		Host:     "smtp.mailgun.org",
		Username: "postmaster@sandbox661c307650f04e909150b37c0f3b2f09.mailgun.org",
		Password: "38304272b8ee5c176d5961dc155b2417",
		Port:     587,
	}
	// change these to your e-mail to check if that works

	// create the service
	mailService := mailer.New(cfg)

	var to = []string{"kataras2006@hotmail.com"}

	// standalone

	//mailService.Send("iris e-mail test subject", "</h1>outside of context before server's listen!</h1>", to...)

	//inside handler
	app.Get("/send", func(ctx *iris.Context) {
		content := `<h1>Hello From Iris web framework</h1> <br/><br/> <span style="color:blue"> This is the rich message body </span>`

		err := mailService.Send("iris e-mail just t3st subject", content, to...)

		if err != nil {
			ctx.HTML(200, "<b> Problem while sending the e-mail: "+err.Error())
		} else {
			ctx.HTML(200, "<h1> SUCCESS </h1>")
		}
	})

	// send a body by template
	app.Get("/send/template", func(ctx *iris.Context) {
		// we will not use ctx.Render
		// because we don't want to render to the client
		// we need the templates' parsed result as raw bytes
		// so we make use of the bytes.Buffer which is an io.Writer
		// which being expected on app.Render parameter first.
		//
		// the rest of the parameters are the same and the behavior is the same as ctx.Render,
		// except the 'where to render'
		buff := &bytes.Buffer{}

		app.Render(buff, "body.html", iris.Map{
			"Message": " his is the rich message body sent by a template!!",
			"Footer":  "The footer of this e-mail!",
		})
		content := buff.String()

		err := mailService.Send("iris e-mail just t3st subject", content, to...)

		if err != nil {
			ctx.HTML(iris.StatusOK, "<b> Problem while sending the e-mail: "+err.Error())
		} else {
			ctx.HTML(iris.StatusOK, "<h1> SUCCESS </h1>")
		}
	})
	app.Listen(":8080")
}
