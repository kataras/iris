//go:build go1.18
// +build go1.18

package main

import (
	"fmt"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/auth"
)

func allowRole(role AccessRole) auth.VerifyUserFunc[User] {
	return func(u User) error {
		if !u.Role.Allow(role) {
			return fmt.Errorf("invalid role")
		}

		return nil
	}
}

const configFilename = "./auth.yml"

func main() {
	app := iris.New()
	app.RegisterView(iris.Blocks(iris.Dir("./views"), ".html").
		LayoutDir("layouts").
		Layout("main"))

	/*
		// Easiest 1-liner way, load from configuration and initialize a new auth instance:
		s := auth.MustLoad[User]("./auth.yml")
		// Bind a configuration from file:
		var c auth.Configuration
		c.BindFile("./auth.yml")
		s, err := auth.New[User](c)
		// OR create new programmatically configuration:
		config := auth.Configuration{
			...fields
		}
		s, err := auth.New[User](config)
		// OR generate a new configuration:
		config := auth.MustGenerateConfiguration()
		s, err := auth.New[User](config)
		// OR generate a new config and save it if cannot open the config file.
		if _, err := os.Stat(configFilename); err != nil {
		    generatedConfig := auth.MustGenerateConfiguration()
		    configContents, err := generatedConfig.ToYAML()
		    if err != nil {
		        panic(err)
		    }

		    err = os.WriteFile(configFilename, configContents, 0600)
		    if err != nil {
		        panic(err)
		    }
		}
	*/

	// 1. Load configuration from a file.
	authConfig, err := auth.LoadConfiguration(configFilename)
	if err != nil {
		panic(err)
	}

	// 2. Initialize a new auth instance for "User" claims (generics: go1.18 +).
	s, err := auth.New[User](authConfig)
	if err != nil {
		panic(err)
	}

	// 3. Add a custom provider, in our case is just a memory-based one.
	s.AddProvider(NewProvider())
	// 3.1. Optionally set a custom error handler.
	// s.SetErrorHandler(new(auth.DefaultErrorHandler))

	app.Get("/signin", renderSigninForm)
	// 4. generate token pairs.
	app.Post("/signin", s.SigninHandler)
	// 5. refresh token pairs.
	app.Post("/refresh", s.RefreshHandler)
	// 6. calls the provider's InvalidateToken method.
	app.Get("/signout", s.SignoutHandler)
	// 7. calls the provider's InvalidateTokens method.
	app.Get("/signout-all", s.SignoutAllHandler)

	// 8.1. allow access for users with "Member" role.
	app.Get("/member", s.VerifyHandler(allowRole(Member)), renderMemberPage(s))
	// 8.2. allow access for users with "Owner" role.
	app.Get("/owner", s.VerifyHandler(allowRole(Owner)), renderOwnerPage(s))

	/* Subdomain user verify:
	app.Subdomain("owner", s.VerifyHandler(allowRole(Owner))).Get("/", renderOwnerPage(s))
	*/
	app.Listen(":8080", iris.WithOptimizations) // Setup HTTPS/TLS for production instead.
	/* Test subdomain user verify, one way is ingrok,
	   add the below to the arguments above:
	, iris.WithConfiguration(iris.Configuration{
		EnableOptmizations: true,
		Tunneling: iris.TunnelingConfiguration{
			AuthToken: "YOUR_AUTH_TOKEN",
			Region:    "us",
			Tunnels: []tunnel.Tunnel{
				{
					Name:     "Iris Auth (Test)",
					Addr:     ":8080",
					Hostname: "YOUR_DOMAIN",
				},
				{
					Name:     "Iris Auth (Test Subdomain)",
					Addr:     ":8080",
					Hostname: "owner.YOUR_DOMAIN",
				},
			},
		},
	})*/
}

func renderSigninForm(ctx iris.Context) {
	if err := ctx.View("signin", iris.Map{"Title": "Signin Page"}); err != nil {
		ctx.HTML("<h3>%s</h3>", err.Error())
		return
	}
}

func renderMemberPage(s *auth.Auth[User]) iris.Handler {
	return func(ctx iris.Context) {
		user := s.GetUser(ctx)
		ctx.Writef("Hello member: %s\n", user.Email)
	}
}

func renderOwnerPage(s *auth.Auth[User]) iris.Handler {
	return func(ctx iris.Context) {
		user := s.GetUser(ctx)
		ctx.Writef("Hello owner: %s\n", user.Email)
	}
}
