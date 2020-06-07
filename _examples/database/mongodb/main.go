package main

// go get -u go.mongodb.org/mongo-driver
// go get -u github.com/joho/godotenv

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	// APIs
	storeapi "myapp/api/store"

	//
	"myapp/env"
	"myapp/store"

	"github.com/kataras/iris/v12"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const version = "0.0.1"

func init() {
	envFileName := ".env"

	flagset := flag.CommandLine
	flagset.StringVar(&envFileName, "env", envFileName, "the env file which web app will use to extract its environment variables")
	flagset.Parse(os.Args[1:])

	env.Load(envFileName)
}

func main() {
	clientOptions := options.Client().SetHosts([]string{env.DSN})
	client, err := mongo.Connect(context.Background(), clientOptions)
	if err != nil {
		log.Fatal(err)
	}

	err = client.Ping(context.Background(), nil)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(context.TODO())

	db := client.Database("store")

	var (
		// Collections.
		moviesCollection = db.Collection("movies")

		// Services.
		movieService = store.NewMovieService(moviesCollection)
	)

	app := iris.New()
	app.Use(func(ctx iris.Context) {
		ctx.Header("Server", "Iris MongoDB/"+version)
		ctx.Next()
	})

	storeAPI := app.Party("/api/store")
	{
		movieHandler := storeapi.NewMovieHandler(movieService)
		storeAPI.Get("/movies", movieHandler.GetAll)
		storeAPI.Post("/movies", movieHandler.Add)
		storeAPI.Get("/movies/{id}", movieHandler.Get)
		storeAPI.Put("/movies/{id}", movieHandler.Update)
		storeAPI.Delete("/movies/{id}", movieHandler.Delete)
	}

	// GET: http://localhost:8080/api/store/movies
	// POST: http://localhost:8080/api/store/movies
	// GET: http://localhost:8080/api/store/movies/{id}
	// PUT: http://localhost:8080/api/store/movies/{id}
	// DELETE: http://localhost:8080/api/store/movies/{id}
	app.Listen(fmt.Sprintf(":%s", env.Port), iris.WithOptimizations)
}
