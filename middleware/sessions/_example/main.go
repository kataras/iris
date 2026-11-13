package main

import (
	"time"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/middleware/sessions"
	"github.com/kataras/iris/v14/rest/errors"
)

// UserSession represents the actual session data for a user.
// You can define any structure that suits your application's needs.
type UserSession struct {
	ID       string
	Username string
}

var (
	// 64-character hash key and 32-character block key.
	// These are used to encrypt and decrypt the session data.
	// You should generate your own random keys for production use.
	hashKey  = []byte("D*G-KaPdSgUkXp2s5v8y/B?E(H+MbQeThWmYq3t6w9z$C&F)J@NcRfUjXn2r4u7x") // 64 bytes length.
	blockKey = []byte("VkYp3s6v9y$B&E)H@McQfTjWmZq4t7w!")                                 // 32 bytes length.
)

func main() {
	app := iris.New()

	sessionOptions := sessions.NewOptions(hashKey, blockKey) // optional encoding settings here.
	sessionOptions.MaxAge = 1 * time.Hour                    // modify the default options here.
	// Cookie-based data example:
	sessionDB := sessions.NewCookieDatabase[UserSession](sessionOptions) // initialize the session database.
	// If you want to use the same cookie name for all sessions (most common usage)
	// then you can use the `Single` function as it doesn't require you to provide
	// a cookie name each time you want to retrieve the session:
	//
	// sessionDB := sessions.Single(
	// 	sessions.NewCookieDatabase[UserSession](sessionOptions),
	// 	"my_session",
	// )
	// app.Get("/set", func(ctx iris.Context) {
	// 	session, err := sessionDB.Get(ctx)
	// 	// [...]
	// }

	/* Redis example:
	import "github.com/redis/go-redis/v9"
	redisClient := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	defer redisClient.Close()
	// [optional check for redisClient.Ping(context.Background()).Val() == "PONG"]
	sessionDB := sessions.NewRedisDatabase[UserSession](redisClient, sessionOptions) // initialize the session database.
	*/

	/* PostgreSQL example:
	import "github.com/jackc/pgx/v5"
	// Initialize the PostgreSQL connection if not already done.
	connString := os.Getenv("DATABASE_URL")
	conn, err := pgx.Connect(context.Background(), connString)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close(context.Background())
	// Initialize the session database.
	sessionDB := sessions.NewPostgresDatabase[UserSession](sessions.DefaultTableName, conn, sessionOptions)
	// Optionally, enable the shift expiration feature.
	// It will update the session's expiration to NOW+Options.MaxAge on each commit.
	// sessions.Handler commits after a request that changed the session, so a request
	// that only reads does not move the expiration: call session.Commit there.
	sessionDB.ShiftExpiration(true)

	// Start the cleanup scheduler, optionally.
	errCh := sessionDB.StartCleanupScheduler(context.Background(), 4*time.Hour) // cleanup expired sessions every 4 hours.
	go func() {
		for err := range errCh {
			fmt.Printf("error during cleanup: %v\n", err)
		}
	}()
	*/

	app.Get("/set", func(ctx iris.Context) {
		session, err := sessionDB.Get(ctx, "my_session")
		if err != nil { // if cookie exists but doesn't exist in database, this will return an error.
			errors.HandleError(ctx, err)
			return
		}

		session.Set(UserSession{
			ID:       "1",
			Username: "kataras",
		})
		if err := session.Commit(ctx); err != nil {
			errors.HandleError(ctx, err)
			return
		}

		ctx.WriteString("session set")
	})

	app.Get("/get", func(ctx iris.Context) {
		session, err := sessionDB.Get(ctx, "my_session")
		if err != nil {
			errors.HandleError(ctx, err)
			return
		}

		ctx.JSON(session.Data)
	})

	app.Get("/delete", func(ctx iris.Context) {
		session, _ := sessionDB.Get(ctx, "my_session")

		session.Options.MaxAge = -1
		if err := session.Commit(ctx); err != nil {
			errors.HandleError(ctx, err)
			return
		} // or session.Delete(ctx)

		ctx.WriteString("session & cookie deleted")
	})

	// flashes.
	app.Get("/set-flash", func(ctx iris.Context) {
		session, err := sessionDB.Get(ctx, "my_session")
		if err != nil {
			errors.HandleError(ctx, err)
			return
		}

		session.SetFlash("notice", "This is a flash message!")
		if err := session.Commit(ctx); err != nil {
			errors.HandleError(ctx, err)
			return
		}

		ctx.WriteString("Flash message set")
	})

	app.Get("/get-flash", func(ctx iris.Context) {
		session, err := sessionDB.Get(ctx, "my_session")
		if err != nil {
			errors.HandleError(ctx, err)
			return
		}

		flash, ok := session.GetFlash[string]("notice") // Retrieve a typed flash message and delete it.
		if ok {
			session.Commit(ctx) // Commit before any server writes, headers should be written before body.
			ctx.WriteString(flash)
		} else {
			ctx.WriteString("No 'notice' flash message")
		}
	})

	app.Get("/get-flashes", func(ctx iris.Context) {
		session, err := sessionDB.Get(ctx, "my_session")
		if err != nil {
			errors.HandleError(ctx, err)
			return
		}

		flashes := session.Flashes(nil) // Retrieve all flash messages without deleting them.
		ctx.JSON(flashes)
	})

	app.Get("/get-and-delete-flashes", func(ctx iris.Context) {
		session, err := sessionDB.Get(ctx, "my_session")
		if err != nil {
			errors.HandleError(ctx, err)
			return
		}

		flashes := session.Flashes(sessions.FlashDelete) // Retrieve all flash messages and delete them.
		if err := session.Commit(ctx); err != nil {      // Commit before any server writes, headers should be written before body.
			errors.HandleError(ctx, err)
			return
		}

		ctx.JSON(flashes)
	})

	// Optional: Usage of Handler and Get functions when you want to use a single session("my_session") for all requests.
	sessionMiddleware := sessions.Handler(sessionDB, "my_session") // see sessions.ReadOnlyHandler too.
	sub := app.Router("/middleware", sessionMiddleware)
	sub.Get("/get", func(ctx iris.Context) {
		session := sessions.Get[UserSession](ctx)
		ctx.JSON(session.Data)
	})
	sub.Get("/update-or-create", func(ctx iris.Context) {
		session := sessions.Get[UserSession](ctx)
		if session.IsNew { // Usage of IsNew.
			session.Data = UserSession{
				ID:       "1",
				Username: "kataras",
			}

			ctx.WriteString("created")
			return
		}

		session.Data.Username = "iris"

		ctx.WriteString("updated")
	})

	app.Listen(":8080")
}
