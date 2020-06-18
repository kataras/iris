package main

import (
	"time"

	"github.com/kataras/iris/v12"

	"github.com/kataras/iris/v12/sessions"
	"github.com/kataras/iris/v12/sessions/sessiondb/redis"

	"github.com/kataras/iris/v12/_examples/sessions/overview/example"
)

// tested with redis version 3.0.503.
// for windows see: https://github.com/ServiceStack/redis-windows
func main() {
	// These are the default values,
	// you can replace them based on your running redis' server settings:
	db := redis.New(redis.Config{
		Network:   "tcp",
		Addr:      "127.0.0.1:6379",
		Timeout:   time.Duration(30) * time.Second,
		MaxActive: 10,
		Password:  "",
		Database:  "",
		Prefix:    "",
		Delim:     "-",
		Driver:    redis.Redigo(), // redis.Radix() can be used instead.
	})

	// Optionally configure the underline driver:
	// driver := redis.Redigo()
	// driver.MaxIdle = ...
	// driver.IdleTimeout = ...
	// driver.Wait = ...
	// redis.Config {Driver: driver}

	// Close connection when control+C/cmd+C
	iris.RegisterOnInterrupt(func() {
		db.Close()
	})

	defer db.Close() // close the database connection if application errored.

	sess := sessions.New(sessions.Config{
		Cookie:          "_session_id",
		Expires:         0, // defaults to 0: unlimited life. Another good value is: 45 * time.Minute,
		AllowReclaim:    true,
		CookieSecureTLS: true,
	})

	//
	// IMPORTANT:
	//
	sess.UseDatabase(db)

	app := example.NewApp(sess)
	app.Listen(":8080")
}
