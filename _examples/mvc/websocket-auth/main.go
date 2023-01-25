//go:build go1.18
// +build go1.18

package main

import (
	"fmt"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/auth"
	"github.com/kataras/iris/v12/mvc"
	"github.com/kataras/iris/v12/websocket"
)

// $ go run .
func main() {
	app := newApp()

	// http://localhost:8080/signin (creds: kataras2006@hotmail.com 123456)
	// http://localhost:8080/protected
	// http://localhost:8080/signout
	app.Listen(":8080")
}

func newApp() *iris.Application {
	app := iris.New()

	// Auth part.
	app.RegisterView(iris.Blocks("./views", ".html").
		LayoutDir("layouts").
		Layout("main"))

	s := auth.MustLoad[User]("./auth.yml")
	s.AddProvider(NewProvider())

	app.Get("/signin", renderSigninForm)
	app.Post("/signin", s.SigninHandler)
	app.Get("/signout", s.SignoutHandler)
	//

	websocketAPI := app.Party("/protected")
	websocketAPI.Use(s.VerifyHandler())
	websocketAPI.HandleDir("/", iris.Dir("./browser")) // render the ./browser/index.html.

	websocketMVC := mvc.New(websocketAPI)
	websocketMVC.HandleWebsocket(new(websocketController))
	websocketServer := websocket.New(websocket.DefaultGorillaUpgrader, websocketMVC)
	websocketAPI.Get("/ws", s.VerifyHandler() /* optional */, websocket.Handler(websocketServer))

	return app
}

func renderSigninForm(ctx iris.Context) {
	if err := ctx.View("signin", iris.Map{"Title": "Signin Page"}); err != nil {
		ctx.HTML("<h3>%s</h3>", err.Error())
		return
	}
}

type websocketController struct {
	*websocket.NSConn `stateless:"true"`
}

func (c *websocketController) Namespace() string {
	return "default"
}

func (c *websocketController) OnChat(msg websocket.Message) error {
	ctx := websocket.GetContext(c.Conn)
	user := auth.GetUser[User](ctx)

	msg.Body = []byte(fmt.Sprintf("%s: %s", user.Email, string(msg.Body)))
	c.Conn.Server().Broadcast(c, msg)

	return nil
}
