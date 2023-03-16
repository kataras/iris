package main

import "github.com/kataras/iris/v12"

func main() {
	app := iris.New()
	app.Post("/", postIndex)

	app.Post("/stream", postIndexStream)

	/*
		curl -L -X POST "http://localhost:8080/" \
		-H 'Content-Type: application/json' \
		--data-raw '{"username":"john"}'

		curl -L -X POST "http://localhost:8080/stream" \
		-H 'Content-Type: application/json' \
		--data-raw '{"username":"john"}
		{"username":"makis"}
		{"username":"george"}
		{"username":"michael"}
		'

		If JSONReader.ArrayStream was true then you must provide an array of objects instead, e.g.
		[{"username":"john"},
		{"username":"makis"},
		{"username":"george"},
		{"username":"michael"}]

	*/

	app.Listen(":8080")
}

type User struct {
	Username string `json:"username"`
}

func postIndex(ctx iris.Context) {
	var u User
	err := ctx.ReadJSON(&u, iris.JSONReader{
		// To throw an error on unknown request payload json fields.
		DisallowUnknownFields: true,
		Optimize:              true,
	})
	if err != nil {
		ctx.StopWithError(iris.StatusBadRequest, err)
		return
	}

	ctx.JSON(iris.Map{
		"code":     iris.StatusOK,
		"username": u.Username,
	})
}

func postIndexStream(ctx iris.Context) {
	var users []User
	job := func(decode iris.DecodeFunc) error {
		var u User
		if err := decode(&u); err != nil {
			return err
		}
		users = append(users, u)
		// When the returned error is not nil the decode operation
		// is terminated and the error is received by the ReadJSONStream method below,
		// otherwise it continues to read the next available object.
		return nil
	}

	err := ctx.ReadJSONStream(job, iris.JSONReader{
		Optimize:              true,
		DisallowUnknownFields: true,
		ArrayStream:           false,
	})
	if err != nil {
		ctx.StopWithError(iris.StatusBadRequest, err)
		return
	}

	ctx.JSON(iris.Map{
		"code":        iris.StatusOK,
		"users_count": len(users),
		"users":       users,
	})
}
