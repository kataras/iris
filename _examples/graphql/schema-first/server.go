package main

import (
	"os"

	"github.com/iris-contrib/outerbanks-api/graph"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/kataras/iris/v12"
)

func main() {
	app := iris.New()

	graphServer := handler.NewDefaultServer(graph.NewExecutableSchema(graph.Config{Resolvers: &graph.Resolver{}}))
	playgroundHandler := playground.Handler("GraphQL playground", "/query")

	app.Get("/", iris.FromStd(playgroundHandler))          // We use iris.FromStd to convert a standard http.Handler to an iris.Handler.
	app.Any("/query", iris.FromStd(graphServer.ServeHTTP)) // GET, POST, PUT...

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	app.Listen(":" + port)
}
