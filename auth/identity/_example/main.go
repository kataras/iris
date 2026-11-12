package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/auth/identity"
	"github.com/kataras/iris/v14/auth/identity/entity"
	"github.com/kataras/iris/v14/auth/identity/password"
	"github.com/kataras/iris/v14/auth/identity/sdk"

	"gopkg.in/yaml.v3"
)

type Configuration struct {
	Identity identity.Options    `yaml:"Identity"`
	Server   serverConfiguration `yaml:"Server"`
}

type serverConfiguration struct {
	iris.Options       `yaml:",inline"` // embed iris.Options to get all the server options.
	iris.ServerOptions `yaml:",inline"` // embed iris.ServerOptions to get all the server options.
	// any custom options here.
}

func loadConfiguration(file string) (cfg Configuration, err error) {
	data, osErr := os.ReadFile(file)
	if osErr != nil {
		return cfg, fmt.Errorf("failed to read configuration file: %w", osErr)
	}

	err = yaml.Unmarshal(data, &cfg)

	return
}

// Run in public: ssh -R 80:localhost:8080 serveo.net.
// Then visit: https://id.hellenic.dev and sign-in using the:
// Generated Admin Token and the Encryption Key (will be shown on the terminal).
// You can also use the generated Server Application Token to call the API using the SDK or any HTTP client.
func main() {
	config, err := loadConfiguration("config.yml")
	if err != nil {
		panic(fmt.Errorf("failed to load configuration: %w", err))
	}

	app := iris.New(config.Server.Options)
	// app.UseRouter(cors.New().
	// 	ExtractOriginFunc(cors.DefaultOriginExtractor).
	// 	ReferrerPolicy(cors.NoReferrerWhenDowngrade).
	// 	HandleErrorFunc(errors.DefaultCORSErrorHandler).
	// 	SkipLocalRequests(config.Server.Addr == ":8080").
	// 	// AllowHeaders("Content-Type", "Content-Length", "Accept-Encoding", "X-CSRF-Token", "Authorization", "Accept", "Origin", "Cache-Control", "X-Requested-With", "X-Token").
	// 	AllowOrigins("https://id.hellenic.dev", "http://localhost:5173"). // this middleware contains the jwks too, I don't know if we must protect it with CORS.
	// 	Handler())

	// identityServer := identity.Load("config.yml")
	identityServer := identity.New(config.Identity)

	serverApp, err := identityServer.AddServerApplication(
		context.Background(),
		"Example-Server-Name",
		"https://www.iris-go.com",
		"https://www.iris-go.com/images/logo.png",
	)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Encryption Key: %s\n", serverApp.EncryptionKey)
	fmt.Printf("Server Application Token: %s\n", serverApp.Token)

	app.Router("/", identityServer)

	// c := client.New(client.BaseURL(opts.BaseURL),
	// 	client.PersistentRequestOptions(client.RequestAuthorizationBearer(opts.Token)),
	// )

	encryptedTestPassword, err := sdk.EncryptPassword("admin!123", serverApp.EncryptionKey)
	if err != nil {
		panic("failed to encrypt password: " + err.Error())
	}
	fmt.Printf("example encrypted password of 'admin!123': %s\n", encryptedTestPassword)

	encryptedTestPassword, err = sdk.EncryptPassword("my-password", serverApp.EncryptionKey)
	if err != nil {
		panic("failed to encrypt password: " + err.Error())
	}
	fmt.Printf("example encrypted password of 'my-password': %s\n", encryptedTestPassword)
	encryptedTestPassword, err = sdk.EncryptPassword("my-password-no-2", serverApp.EncryptionKey)
	if err != nil {
		panic("failed to encrypt password: " + err.Error())
	}
	fmt.Printf("example encrypted password of 'my-password-no-2': %s\n", encryptedTestPassword)

	rawPass := "@nN123456"
	encryptedTestPassword, err = sdk.EncryptPassword(rawPass, serverApp.EncryptionKey)
	if err != nil {
		panic("failed to encrypt password: " + err.Error())
	}
	fmt.Printf("example encrypted password of '%s': %s\n", rawPass, encryptedTestPassword)

	gotRawPass, err := password.Decrypt(encryptedTestPassword, serverApp.EncryptionKey) // just to test the decrypt function.
	if err != nil {
		panic("failed to decrypt password: " + err.Error())
	}
	fmt.Printf("decrypted password: %s\n", gotRawPass)
	if rawPass != gotRawPass {
		panic(fmt.Sprintf("decrypt password does not match the original one: %s vs %s", rawPass, gotRawPass))
	}

	// let's create a user from the identity server itself,
	// in real world this should be done via a registration form and the SDK which provides a typed way of doing this.
	newUser := entity.User{
		Username: "kataras2006",
		Password: encryptedTestPassword,
		Attrs: json.RawMessage(`{
            "phone": "+306975807921",
            "firstname": "Makis",
            "lastname": "Maropoulos",
            "birthdate": "1990-03-15"
			}`),
	}
	_, _, err = identityServer.Provider.Signup(context.Background(), serverApp.Client, newUser)
	if err != nil {
		if !errors.Is(err, entity.ErrAlreadyExists) {
			panic(fmt.Errorf("failed to signup a test user: %w", err))
		}
	}

	/*
		usersPage, totalCount, err := identityServer.Provider.ListUsersPaginatedFlattened(
			context.Background(),
			entity.PageOptions{
				Size: 10,
			},
			&entity.UserFilterOptions{
				Sort:           "created_at",
				SortDescending: true,
				IncludeDeleted: true,
				Terms: []*entity.FilterTerm{
					// {Field: "username", Operator: "ILIKE", Value: "%test%"},
					// {Field: "attrs->>'key91'", Operator: "=", Value: "148"},
					// {Logic: "OR", Field: "attrs->>'key91'", Operator: "=", Value: "148"},
					// {Field: "deleted_at", Operator: "IS NOT", Value: "NULL"},
					{Field: "email", Operator: "ILIKE", Value: "%example.com%"},
				},
			})
		if err != nil {
			panic(fmt.Errorf("list users: %w", err))
		}

		fmt.Printf("found %d total users\n", totalCount)
		for i, user := range usersPage {
			b, err := json.MarshalIndent(user, "", "    ")
			if err != nil {
				panic(fmt.Errorf("failed to marshal user %d: %w", i, err))
			}
			fmt.Printf("[%d] User: %s\n\n", i, string(b))
		}
	*/

	app.Run(config.Server.ServerOptions)
}
