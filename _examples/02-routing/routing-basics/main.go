// Routing basics: one function per HTTP method, and the Context helpers that
// carry the status code and the error branch for you.
//
// ctx.BindJSON decodes a body and writes the 400 itself, ctx.List renders a
// slice as a JSON array, ctx.Created sends 201 with an id payload, ctx.OK
// sends 200 and ctx.NoContent sends 204. errors.NotFoundID renders the 404.
// None of them needs a hand-written status code or an error map at the call
// site.
//
// The store is a map behind a mutex so the write methods do something real;
// the point of the example is the route table and the response calls.
//
// Learn: register one handler per HTTP method and answer with ctx.OK, ctx.Created,
// ctx.NoContent and ctx.List instead of hand-written status codes.
// Before: ../../01-getting-started
//
// Run:
//
//	go run main.go
//
// Try:
//
//	curl http://localhost:8080/ping                                # 200, pong
//	curl http://localhost:8080/tasks                               # 200, []
//	curl -X POST http://localhost:8080/tasks -H 'Content-Type: application/json' -d '{"title":"write docs"}'          # 201, {"id":1}
//	curl -X PUT http://localhost:8080/tasks/1 -H 'Content-Type: application/json' -d '{"title":"write better docs"}'  # 200, the updated task
//	curl -i -X DELETE http://localhost:8080/tasks/1                # 204, no body
//	curl -i -X DELETE http://localhost:8080/tasks/1                # 404, again: not found envelope
//
// Next: ../path-parameters
package main

import (
	"sync"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/rest/errors"
)

func main() {
	app := iris.New()

	// Each method function (Get, Post, Put, Delete, Patch, Head, Options...)
	// takes a path pattern and one or more handlers.
	app.Get("/ping", ping)
	app.Get("/tasks", listTasks)
	app.Post("/tasks", createTask)
	app.Put("/tasks/{id:int}", updateTask)
	app.Delete("/tasks/{id:int}", deleteTask)

	app.Listen(":8080")
}

type task struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
}

var (
	mu     sync.Mutex
	nextID = 1
	tasks  = map[int]task{}
)

// ctx.WriteString writes a plain string body; nothing is serialized and no
// JSON content type is set.
func ping(ctx iris.Context) {
	ctx.WriteString("pong")
}

func listTasks(ctx iris.Context) {
	mu.Lock()
	defer mu.Unlock()

	list := make([]task, 0, len(tasks))
	for _, t := range tasks {
		list = append(list, t)
	}

	// ctx.List renders a slice as a 200 JSON array and an empty one as [],
	// never null, so the client can always iterate the answer.
	ctx.List(list, nil)
}

func createTask(ctx iris.Context) {
	// ctx.BindJSON decodes the body and, on a malformed one, writes the 400
	// itself and returns false. The handler only ever sees decoded input.
	input, ok := ctx.BindJSON[task]()
	if !ok {
		return
	}

	mu.Lock()
	input.ID = nextID
	nextID++
	tasks[input.ID] = input
	mu.Unlock()

	// ctx.Created sends 201 and wraps a bare identifier as {"id": ...}.
	ctx.Created(input.ID, nil)
}

func updateTask(ctx iris.Context) {
	// The {id:int} macro already validated and converted the segment.
	id := ctx.Params().GetInt("id")

	input, ok := ctx.BindJSON[task]()
	if !ok {
		return
	}

	mu.Lock()
	defer mu.Unlock()

	if _, ok := tasks[id]; !ok {
		// errors.NotFoundID renders the 404 error envelope naming the
		// resource and the id that was not there.
		errors.NotFoundID(ctx, "task", id)
		return
	}

	input.ID = id
	tasks[id] = input
	// ctx.OK is the 200 half of the same pair: a nil error sends the value.
	ctx.OK(input, nil)
}

func deleteTask(ctx iris.Context) {
	id := ctx.Params().GetInt("id")

	mu.Lock()
	defer mu.Unlock()

	if _, ok := tasks[id]; !ok {
		errors.NotFoundID(ctx, "task", id)
		return
	}

	delete(tasks, id)
	// ctx.NoContent sends 204 for a delete that changed something and 304
	// for one that did not; either way there is no body.
	ctx.NoContent(true, nil)
}
