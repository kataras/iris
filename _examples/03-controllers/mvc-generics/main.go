// Generic MVC controllers: five typed methods, the REST routes generated.
//
// mvc.UseController[*T]() reads the method signatures of a type that satisfies
// mvc.Controller and registers the CRUD routes for it. Iris decodes the body
// into the Request type, converts the path id to the ID type, encodes the
// Response and installs the rest/errors handler on the child router. Your
// methods only return (value, error).
//
// The generated path parameter is named after the Response model (Task becomes
// task_id) and its macro comes from the ID type: int, int64 and uint64 map to
// the matching macro, uuid.UUID maps to :uuid, and every other ID type, string
// included, stays :string. Declare the controller with uuid.UUID rather than
// string to have the router reject a malformed id before any handler runs.
// mvc.GetPathParam hands you that name inside the optional Routing method.
//
// The controller itself comes from the container: it is resolved by type, or
// constructed and its exported fields bound from the registered dependencies,
// which is what happens to Store below.
//
// Generated routes:
//
//	POST   /tasks                          -> Create, 201
//	GET    /tasks/{task_id:string}         -> Get
//	PUT    /tasks/{task_id:string}?columns -> Update, 200 (or 204/304 for a nil response)
//	DELETE /tasks/{task_id:string}         -> Delete, 204/304
//	POST   /tasks/list                     -> List, paginated
//
// The generated handlers pick the decoder from the request's Content-Type, so
// a JSON body needs the header, unlike ctx.BindJSON, which always reads JSON.
//
// Learn: have the CRUD routes of a resource generated from five typed
// controller methods with mvc.UseController, and add extra routes beside them.
// Before: ../nested-guards
//
// Run:
//
//	go run main.go
//
// Try:
//
//	H='Content-Type: application/json'
//	curl -H "$H" -X POST http://localhost:8080/tasks -d '{"title":"write docs"}'                    # 201, the task with its id
//	curl -H "$H" -X POST http://localhost:8080/tasks/list -d '{"done":false}'                       # 200, the Page envelope
//	curl http://localhost:8080/tasks/<the id from create>                                           # 200, the task
//	curl -H "$H" -X PUT 'http://localhost:8080/tasks/<id>?columns=done' -d '{"done":true}'          # 200, the updated task
//	curl -i -X DELETE http://localhost:8080/tasks/<id>                                              # 204, then 304 on a second call
//	curl http://localhost:8080/tasks/param                                                          # 200, {"name":"task_id","type":"string"}
//
// Next: ../repository
package main

import (
	"fmt"
	"log"
	"sync"

	"uuid"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/controller/mvc"
	"github.com/kataras/iris/v14/rest/errors"
)

func main() {
	app := iris.New()

	if err := app.RegisterDependency(NewTaskStore()); err != nil {
		log.Fatal(err)
	}

	app.Router("/tasks", mvc.UseController[*TasksController]())

	app.Listen(":8080")
}

// TaskStore is a dependency, registered before the controller is mounted.
type TaskStore struct {
	mu    sync.RWMutex
	tasks map[string]Task
}

func NewTaskStore() *TaskStore {
	return &TaskStore{tasks: make(map[string]Task)}
}

// TasksController implements mvc.Controller[iris.Context, Task, Task, string,
// TaskFilter, int]. The type arguments are inferred from these five methods.
type TasksController struct {
	Store *TaskStore // exported: bound from the registered *TaskStore.
}

// POST /tasks
func (c *TasksController) Create(ctx iris.Context, in Task) (Task, error) {
	in.ID = uuid.New().String()

	c.Store.mu.Lock()
	c.Store.tasks[in.ID] = in
	c.Store.mu.Unlock()

	return in, nil
}

// GET /tasks/{task_id:string}
func (c *TasksController) Get(ctx iris.Context, id string) (Task, error) {
	c.Store.mu.RLock()
	task, ok := c.Store.tasks[id]
	c.Store.mu.RUnlock()

	if !ok {
		// Returned errors flow through the rest/errors pipeline: 404 here.
		return Task{}, errors.NotFound.Wrap(fmt.Errorf("task %s not found", id))
	}

	return task, nil
}

// PUT /tasks/{task_id:string}?columns=done
// A non-nil *Task renders 200; a nil one lets the boolean pick: true is 204
// No Content, false is 304 Not Modified.
func (c *TasksController) Update(ctx iris.Context, id string, in Task,
	columns ...string) (*Task, bool, error) {
	c.Store.mu.Lock()
	defer c.Store.mu.Unlock()

	task, ok := c.Store.tasks[id]
	if !ok {
		return nil, false, errors.NotFound.Wrap(fmt.Errorf("task %s not found", id))
	}

	// "columns" holds the ?columns= values: the fields the client asked to
	// change. An empty list means "everything".
	for _, column := range orAll(columns) {
		switch column {
		case "title":
			task.Title = in.Title
		case "done":
			task.Done = in.Done
		}
	}

	c.Store.tasks[id] = task
	return &task, true, nil
}

// DELETE /tasks/{task_id:string}
func (c *TasksController) Delete(ctx iris.Context, id string) (bool, error) {
	c.Store.mu.Lock()
	defer c.Store.mu.Unlock()

	if _, ok := c.Store.tasks[id]; !ok {
		return false, nil // 304: nothing was there to delete.
	}

	delete(c.Store.tasks, id)
	return true, nil // 204.
}

// POST /tasks/list
func (c *TasksController) List(ctx iris.Context, filter TaskFilter,
	limit, offset int) ([]Task, int, error) {
	c.Store.mu.RLock()
	defer c.Store.mu.RUnlock()

	var matched []Task
	for _, task := range c.Store.tasks {
		if task.Done == filter.Done {
			matched = append(matched, task)
		}
	}

	total := len(matched)
	if offset >= total {
		return nil, total, nil
	}
	if end := offset + limit; limit > 0 && end < total {
		matched = matched[offset:end]
	} else {
		matched = matched[offset:]
	}

	return matched, total, nil
}

// Routing is optional: it adds routes beside the generated ones.
func (c *TasksController) Routing(r iris.Router) {
	param := mvc.GetPathParam(c) // the generated {task_id:string}.

	r.Get("/param", func(ctx iris.Context) {
		ctx.JSON(iris.Map{"name": param.Name, "type": param.Type})
	})
}

// Task is both the Request and the Response type of the controller.
type Task struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Done  bool   `json:"done"`
}

// TaskFilter is the List payload, read from the POST /tasks/list body.
type TaskFilter struct {
	Done bool `json:"done"`
}

func orAll(columns []string) []string {
	if len(columns) == 0 {
		return []string{"title", "done"}
	}

	return columns
}
