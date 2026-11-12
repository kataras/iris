# MVC

## The MVC pattern

Model-View-Controller splits an application into three parts, each responsible for one concern:

- **Model**: Represents the data and the business logic of the application.
- **View**: Represents the UI components and displays the data from the model to the user. In the world of [RESTful](https://en.wikipedia.org/wiki/REST), the `View` is the JSON response.
- **Controller**: Handles user input and updates the model and view accordingly.

For more information on MVC, you can refer to the [Wikipedia page on MVC](https://en.wikipedia.org/wiki/Model%E2%80%93view%E2%80%93controller).

## Iris MVC

Iris implements MVC on top of Go generics. A controller is your own type: implement `Create`, `Get`, `Update`, `Delete` and `List` and `mvc.UseController` registers the matching CRUD routes for you, resolving the controller and its dependencies from the container. A method's return values become the response and its error goes through the central error map. Add a `Routing` method when you need routes beyond CRUD.

### Features of Iris MVC

1. **Automatic Route Generation**: Automatically generates routes for CRUD operations.
2. **Dependency Injection**: Easily inject dependencies into controllers.
3. **Custom Routing**: Define custom routes for controllers.
4. **Path Parameters**: Automatically handle path parameters in routes.
5. **View Support**: Render views with specific models.
6. **WebSocket Support**: Handle WebSocket connections and events in a structured manner.
7. **Flexible and Extensible**: Easily extend and customize the framework to fit your needs.

### Installation

To use the Iris MVC, you need to have the Iris framework installed. You can install Iris using the following command:

```sh
go get github.com/kataras/iris/v14@latest
```

### Basic Setup

To use the MVC, you need to import the necessary packages and configure the controller in your Iris application. Here is a basic example:

```go
package main

import (
	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/controller/mvc"
)

func main() {
	app := iris.New()
	app.Router("/stores", mvc.UseController[*StoreController]())

	app.Listen(":8080")
}
```

### Defining a Controller

A controller in Iris MVC is a struct that implements the `Controller` interface. The `Controller` interface defines the methods for CRUD operations:

```go
type Controller[Context handler.ContextType, Request any, Response any, ID context.IDType, Filter any, Total context.CountType] interface {
	// POST: /resource
	Create(Context, Request) (Response, error)
	// GET: /resource/{id}
	Get(Context, ID) (Response, error)
	// PUT: /resource/{id}?columns=column1&columns=column2
	Update(Context, ID, Request, ...string) (*Response, bool, error)
	// DELETE: /resource/{id}
	Delete(Context, ID) (bool, error)
	// POST: /resource/list
	List(ctx Context, filter Filter, limit int, offset int) ([]Response, Total, error)
}
```

### CRUD Methods in the Controller Interface

#### Create

```go
Create(Context, Request) (Response, error)
```

- **Description**: Handles the creation of a new resource.
- **Response Handling**:
  - Response is written as a JSON response with a status code of `201 Created`.
- **Error Handling**: If an error is returned, it is automatically handled by the `errors.HandleError` function, which sends an appropriate error response to the client.

#### Get

```go
Get(Context, ID) (Response, error)
```

- **Description**: Handles retrieving a resource by its ID.
- **Response Handling**:
  - Response is written as a JSON response with a status code of `200 OK`.
- **Error Handling**: If an error is returned, it is automatically handled by the `errors.HandleError` function, which sends an appropriate error response to the client.

#### Update

```go
Update(Context, ID, Request, ...string) (*Response, bool, error)
```

- **Description**: Handles updating a resource by its ID. The variadic strings are the `?columns=` query values: the fields the client asked to change.
- **Response Handling**:
  - If the returned `Response` is not `nil`, it is written as a JSON response with a status code of `200 OK`.
  - If the `Response` is `nil`, the boolean decides: `true` is `204 No Content` (the update changed something), `false` is `304 Not Modified` (it changed nothing). Both bodies are empty.
- **Error Handling**: If an error is returned, it is automatically handled by the `errors.HandleError` function, which sends an appropriate error response to the client.

#### Delete

```go
Delete(Context, ID) (bool, error)
```

- **Description**: Handles deleting a resource by its ID.
- **Response Handling**:
  - If the boolean return value is `true`, the status code is `204 No Content`.
  - If the boolean return value is `false`, nothing was there to delete: the status code is `304 Not Modified`, with an empty body.
- **Error Handling**: If an error is returned, it is automatically handled by the `errors.HandleError` function, which sends an appropriate error response to the client.

#### List

```go
List(ctx Context, filter Filter, limit int, offset int) ([]Response, Total, error)
```

- **Description**: Handles listing resources with optional filtering, pagination, and sorting.
- **Response Handling**:
  - The returned list of `Response` items is written as an `iris.Page` JSON response with a status code of `200 OK`.
- **Error Handling**: If an error is returned, it is automatically handled by the `errors.HandleError` function, which sends an appropriate error response to the client.


### Registering a Controller

To register a controller, use the `UseController` function. This function sets up the necessary routes for CRUD operations and returns a `DeployedController` which can be used to add additional routes or controllers under the same path.

```go
app.Router("/stores", mvc.UseController[*StoreController]())
```

### Dependency Injection

To inject dependencies into your controller, you need to register them with the Iris application. Use the `RegisterDependency` method to register your services. The framework will automatically bind the registered dependencies to the controller's fields.

Here is an example of how to register a `StoreService`:

```go
app.RegisterDependency(NewStoreService)
```

### Custom Routing

Implement the `Routing` method to register routes beyond the generated CRUD set.

```go
func (c *StoreController) Routing(r iris.Router) {
	r.Get("/custom", func(ctx iris.Context) {
		ctx.WriteString("This is a custom route")
	})
}
```

### Using the `With` Functionality

`DeployedController.With` mounts another controller under the same path, which is how nested resources such as `/stores/{store_id}/books` are built.

Here is an example of how to use the `With` method:

```go
app.Router("/stores", mvc.UseController[*StoreController]().
	With("/books", mvc.UseController[*BookController]()))
```

In this example, the `BookController` is registered under the `/stores/{store_id}/books` path, creating a nested route structure.

### Using `mvc.GetPathParam`

`mvc.GetPathParam` returns the controller's path parameter, for use in a custom route or anywhere else in the controller.

Here is an example of how to use `mvc.GetPathParam`:

```go
func (c *StoreController) Routing(r iris.Router) {
	param := mvc.GetPathParam(c)
	r.Get("/greet", func(ctx iris.Context) {
		ctx.Writef("%s (%s)", c.Service.Greet(), ctx.Params().GetString(param.Name))
	})
}
```

## Repository-backed Resources and Options

Some resources add nothing to their repository: the HTTP half of the controller decodes a body, calls one method and renders what came back. `mvc.UseRepository` registers the routes straight over the repository, with no controller type in between.

A repository is the five methods your service already has, over the standard context. It never sees Iris:

```go
type Repository[T any, ID context.IDType, Filter any] interface {
	Create(ctx context.Context, v T) (ID, error)
	Get(ctx context.Context, id ID) (T, error)
	Update(ctx context.Context, id ID, v T, onlyColumns ...string) (updated bool, err error)
	Delete(ctx context.Context, id ID) (deleted bool, err error)
	List(ctx context.Context, filter Filter, limit, offset int) ([]T, int64, error)
}
```

Register it the same way a controller is registered:

```go
app.Router("/invoices", mvc.UseRepository(invoiceRepository))
```

That generates six routes:

| Method and path | Response |
| --------------- | -------- |
| `POST /invoices` | 201 `{"id": ...}` |
| `GET /invoices/{invoice_id}` | 200, the model |
| `PUT /invoices/{invoice_id}` | 204, or 304 when nothing changed |
| `PATCH /invoices/{invoice_id}?columns=a&columns=b` | the same, limited to the named columns |
| `DELETE /invoices/{invoice_id}` | 204, or 304 when nothing was deleted |
| `POST /invoices/list` | 200, a page of models |

The path parameter name and its macro derive from the model and its ID type exactly as they do for `mvc.UseController`, so a `uuid.UUID` id registers `{invoice_id:uuid}` and validates before any handler runs. Errors from the repository render through the central error map.

The two update routes read their columns through `Context.BindUpdate`, which accepts both spellings (`?columns=a&columns=b` and `?columns=a,b`), trims and de-duplicates them, and answers 400 for a name that is not a member of the model, so a typo is reported instead of silently updating nothing. The names that reach `onlyColumns` are still the client's: map them to your own columns in the repository and never interpolate them into SQL.

Reach for a controller instead as soon as a route needs anything the repository does not do: a mail on create, a computed response, a second store to keep in sync.

### Options

The same options apply to `mvc.UseController` and to `mvc.UseRepository`:

| Option | Description |
| ------ | ----------- |
| `mvc.Guard(handlers ...iris.Handler)` | Middleware for the write routes only: Create, Update and Delete. Reads stay as they were, which is the shape a router-level `Use` cannot express without splitting the resource in two. |
| `mvc.Without(verbs ...mvc.Verb)` | Drops generated routes. The verbs are `mvc.Create`, `mvc.Get`, `mvc.Update`, `mvc.Delete` and `mvc.List`. |
| `mvc.IDParam(name string)` | Overrides the derived parameter name, so `/customers/{customer_id}` can be registered as `/customers/{id}`. The macro still comes from the ID type. |
| `mvc.BeforeWrite(fn func(ctx iris.Context, v *T) error)` | Runs on Create and Update, after the body is decoded and before the repository or the controller sees it. A non-nil error stops the request and renders through the error pipeline. Hooks run in registration order. A hook whose `T` is not the type the resource writes panics when the routes are registered, naming both types. |
| `mvc.Pagination(opts ...iris.PaginationOption)` | Declares the List route's sort policy: `iris.AllowSort("created_at", "name")` limits the column a client may sort by and `iris.DenySort("...")` refuses any sort, both answering 400 before the controller or the repository runs. The policy reads the decoded filter through its `SortRequest` method, which `iris.ListFilter` provides. |

`BeforeWrite` is how a value the client must not choose is stamped on every body, in one place instead of in every handler:

`auth.Scope` belongs on the router, so that the read routes are scoped too; `mvc.Guard` carries only what the write routes need on top of it.

```go
app.Use(a.VerifyHandler(), auth.Scope(func(u Customer) string { return u.TenantID }))

app.Router("/invoices", mvc.UseRepository(invoiceRepository,
	mvc.Guard(requireWriteRole),
	mvc.Pagination(iris.AllowSort("created_at")),
	mvc.BeforeWrite(func(ctx iris.Context, v *Invoice) error {
		tenantID, ok := auth.ScopeID[string](ctx)
		if !ok {
			return errors.Unauthenticated
		}

		v.TenantID = tenantID
		return nil
	}),
))
```

The model type is a type parameter, so the hook receives `*Invoice` and no reflection is involved at serve time.

## Auto-Binding of Path Parameters

The Iris MVC supports auto-binding of path parameters to the controller's fields. This is particularly useful when you have nested routes and need to access the parent resource's ID in the child controller.

### Using `mvc.PathParam`

The `mvc.PathParam[T, V]` type is a dynamic typed path parameter which can be used as a dependency to the controller's fields. `T` is the Response type and `V` is the ID type (`string`, `int`, `int64`, `uint64` or `uuid.UUID`).

Here is an example of how to use `mvc.PathParam` in a `BookController`:

```go
// Model
type Book struct {
	ID      string `json:"id"`
	StoreID string `json:"store_id"`
	Title   string `json:"title"`
	Author  string `json:"author"`
}

// Service
type BookService struct{}

func NewBookService() *BookService {
	return &BookService{}
}

func (s *BookService) Greet() string {
	return "Hello, Book!"
}

// Controller
type BookController struct {
	Service    *BookService
	StoreParam *mvc.PathParam[Store, string]
}

func (c *BookController) Create(ctx iris.Context, book Book) (Book, error) {
	book.StoreID = c.StoreParam.Get(ctx)
	return book, nil
}

func (c *BookController) Get(ctx iris.Context, id string) (Book, error) {
	return Book{ID: id, StoreID: c.StoreParam.Get(ctx), Title: "The Catcher in the Rye", Author: "J.D. Salinger"}, nil
}
```

In this example, the `BookController` has a `StoreParam` field of type `*mvc.PathParam[Store, string]`. This field is automatically bound to the path parameter of the parent `StoreController`. The `StoreParam.Get(ctx)` method is used to retrieve the actual value of the path parameter from the context.

### Example Usage

Here is a complete example that demonstrates how to set up and use the Iris MVC with auto-binding of path parameters:

```go
package main

import (
	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/controller/mvc"
)

// Model
type Store struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Location string `json:"location"`
}

// Service
type StoreService struct{}

func NewStoreService() *StoreService {
	return &StoreService{}
}

func (s *StoreService) Greet() string {
	return "Hello, Store!"
}

// Controller
type StoreController struct {
	Service *StoreService
}

func (c *StoreController) Create(ctx iris.Context, store Store) (Store, error) {
	return store, nil
}

func (c *StoreController) Get(ctx iris.Context, id string) (Store, error) {
	return Store{ID: id, Name: "Bookstore", Location: "New York"}, nil
}

func (c *StoreController) Update(ctx iris.Context, id string, store Store, onlyColumns ...string) (*Store, bool, error) {
	return &store, true, nil
}

func (c *StoreController) Delete(ctx iris.Context, id string) (bool, error) {
	return true, nil
}

func (c *StoreController) List(ctx iris.Context, filter StoreFilter, limit int, offset int) ([]Store, int, error) {
	return []Store{{ID: "a6ce7330-d584-4675-af9e-2c2d2c40de07", Name: "Bookstore", Location: "New York"}}, 1, nil
}

func (c *StoreController) Other(ctx iris.Context) {
    ctx.WriteString("This is another route")
}

func (c *StoreController) Routing(r iris.Router) {
	param := mvc.GetPathParam(c)

	r.Get("/greet", func(ctx iris.Context) {
		ctx.Writef("%s (%s)", c.Service.Greet(), ctx.Params().GetString(param.Name))
	})

    // Another custom route as a controller's method.
    r.Get("/other", c.Other)
}

// Filter Model
type StoreFilter struct {
	Location string `json:"location"`
}

// Model
type Book struct {
	ID      string `json:"id"`
	StoreID string `json:"store_id"`
	Title   string `json:"title"`
	Author  string `json:"author"`
}

// Service
type BookService struct{}

func NewBookService() *BookService {
	return &BookService{}
}

func (s *BookService) Greet() string {
	return "Hello, Book!"
}

// Controller
type BookController struct {
	Service    *BookService
	StoreParam *mvc.PathParam[Store, string]
}

func (c *BookController) Create(ctx iris.Context, book Book) (Book, error) {
    storeID := c.StoreParam.Get(ctx) // Get the path parameter from the context.

	book.StoreID = storeID
	return book, nil
}

func (c *BookController) Get(ctx iris.Context, id string) (Book, error) {
    storeID := c.StoreParam.Get(ctx)

	return Book{ID: id, StoreID: storeID, Title: "The Catcher in the Rye", Author: "J.D. Salinger"}, nil
}

func (c *BookController) Update(ctx iris.Context, id string, book Book, onlyColumns ...string) (*Book, bool, error) {
	book.ID = id
	book.StoreID = c.StoreParam.Get(ctx)
	return &book, true, nil
}

func (c *BookController) Delete(ctx iris.Context, id string) (bool, error) {
	return true, nil
}

func (c *BookController) List(ctx iris.Context, filter BookFilter, limit int, offset int) ([]Book, int, error) {
	storeID := c.StoreParam.Get(ctx)
	return []Book{{ID: "1", StoreID: storeID, Title: "The Catcher in the Rye", Author: "J.D. Salinger"}}, 1, nil
}

// Filter Model
type BookFilter struct {
	Author string `json:"author"`
}

func main() {
	app := iris.New()
	app.RegisterDependency(NewStoreService)
	app.RegisterDependency(NewBookService)

	app.Router("/stores", mvc.UseController[*StoreController]().
		With("/books", mvc.UseController[*BookController]()))

	app.Listen(":8080")
}
```

In this example, we define a `Store` model, a `StoreService`, and a `StoreController`. We also define a `Book` model, a `BookService`, and a `BookController`. We register the `StoreController` with the Iris application and set up the necessary routes for CRUD operations. Additionally, we use the `With` method to register a `BookController` under the `/stores/{store_id}/books` path, creating a nested route structure. We also demonstrate how to use `mvc.GetPathParam` to access the path parameter in a custom route and how to use `mvc.PathParam` for auto-binding of path parameters.

### Using ViewModel as a Response

Return a `ViewModel` from a controller method to render a template with that model instead of serializing the value.

#### ViewModel Structure

The `ViewModel` structure is a generic type that provides a way to render views with a specific model.

```go
type ViewModel[T any] struct {...}
```

#### Creating a ViewModel

To create a `ViewModel`, use the `View` function. This function accepts the status code, layout, view name, and model data as parameters.

```go
func View[T any](status int, layout, name string, data T) ViewModel[T]
```


#### Example Usage

First, register the view engine you want to use, e.g. [blocks](https://github.com/kataras/blocks), through the `view` middleware:

```go
import (
	"github.com/kataras/blocks"
	"github.com/kataras/iris/v14/middleware/view"
)

viewEngine := blocks.New("./views")
app.UseRouter(view.Handler(viewEngine))
```

Here is an example of how to use the `ViewModel` structure in a controller method:

```go
func (c *BookViewController) Get(ctx iris.Context, id string) (mvc.ViewModel[Book], error) {
    storeID := c.StoreParam.Get(ctx)

	book := Book{ID: id, StoreID: storeID, Title: "The Catcher in the Rye", Author: "J.D. Salinger"}

	return mvc.View(iris.StatusOK, "main", "book.html", book), nil
}
```

### Extending Controllers with View Rendering Capabilities

`WithViews` adds view rendering to a controller, for the endpoints that answer with HTML rather than JSON.

#### Using the `WithViews` Method

The `WithViews` method extends a deployed controller with view rendering capabilities. It registers routes for various view-related endpoints and binds them to the corresponding controller methods. It is generic over the four view data types, and the controller must implement the `ViewController` interface for them.

Here is an example of how to use the `WithViews` method:

```go
app.Router("/books", mvc.UseController[*BookController]().WithViews[IndexViewData, ViewData, ViewData, ListViewData]())
```

Here's how you can restrucure the `BookController` to support view rendering:

> Note that the specified templates in the documents must be already loaded using the `view.Handler` middleware.

```go
type (
	IndexViewData struct {
		Name string
	}

	ViewData struct { 
	// Could be the same for all view methods, but we separate them for the
	// shake of the example
		Message string
	}

	ListViewData struct {
		RootMessage string

		Data []ViewData
	}
)

// GET: /resource renders the book/index.html template.
func (c *BookController) IndexView(ctx iris.Context) (IndexViewData, error) {
	return IndexViewData{Name: "Index!"}, nil
}

// GET: /resource/create renders the book/create.html template.
func (c *BookController) CreateView(ctx iris.Context) (ViewData, error) {
	return ViewData{Message: "Create, View!"}, nil
}

// GET: /resource/{id}/details renders the book/details.html template.
func (c *BookController) DetailsView(ctx iris.Context, id string) (ViewData, error) {
	return ViewData{Message: "Get, Details View!"}, nil
}

// GET: /resource/{id}/update renders the book/update.html template.
func (c *BookController) UpdateView(ctx iris.Context, id string) (ViewData, error) {
	return ViewData{Message: "Update Message!"}, nil
}

// GET: /resource/{id}/delete renders the book/delete.html template
func (c *BookController) DeleteView(ctx iris.Context, id string) (ViewData, error) {
	return ViewData{Message: "Delete Message!"}, nil
}

// GET: /resource/list renders the book/list.html template.
func (c *BookController) ListView(ctx iris.Context) (ListViewData, error) {
	return ListViewData{RootMessage: "Hello, World from Root List View Data Object!", Data: []ViewData{{Message: "Hello, World!"}}}, nil
}
```

### WebSocket Support

A controller can also serve WebSocket connections, with one method per event.

```sh
go get github.com/kataras/neffos@latest
```

#### Defining a WebSocket Controller

A WebSocket controller in Iris MVC is a struct that implements methods to handle [neffos](https://github.com/kataras/neffos) WebSocket events.

#### Registering a WebSocket Controller

To register a WebSocket controller, use the `mvc.Websocket` function. This function sets up the necessary routes for WebSocket connections and events.

Here is an example of how to register a `websocketController`:

```go
package main

import (
	"fmt"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/controller/fileserver"
	"github.com/kataras/iris/v14/controller/mvc"
	"github.com/kataras/iris/v14/middleware/counter"
	"github.com/kataras/iris/v14/middleware/websocket"

	"github.com/kataras/golog"
	"github.com/kataras/neffos"
)

func main() {
	app := iris.New()

	// Render the ./browser/index.html.
	app.Router("/", fileserver.New("./browser", fileserver.DefaultOptions))

	// Register a dependency which will be used by the WebSocket controller.
	app.RegisterDependency(app.Logger, newTestService)

	// Register the WebSocket server and controller(s).
	app.Router("/websocket", mvc.Websocket(newWebsocketController))

	// Start the server.
	app.Listen(":8080")
}

type testService struct {
	logger *golog.Logger
}

func newTestService(logger *golog.Logger) *testService {
	return &testService{
		logger: logger,
	}
}

func (s *testService) Log(format string, args ...any) {
	s.logger.Infof(format, args...)
}

type websocketController struct {
	service *testService
	counter *counter.Counter[int]
}

func newWebsocketController(service *testService) *websocketController {
	return &websocketController{
		service: service,
		counter: counter.NewCounter[int](),
	}
}

func (c *websocketController) Namespace() string {
	return "default"
}

func (c *websocketController) OnNamespaceConnected(conn *neffos.NSConn, msg neffos.Message) error {
	newCount := c.counter.Increment()
	c.service.Log("Connected [%d]", newCount)

	conn.Broadcast(neffos.Message{
		Namespace: msg.Namespace,
		Event:     "OnVisit",
		Body:      []byte(fmt.Sprintf("%d", newCount)),
	})

	return nil
}

func (c *websocketController) OnNamespaceDisconnect(conn *neffos.NSConn, msg neffos.Message) error {
	newCount := c.counter.Decrement()
	c.service.Log("Disconnected (%d)", newCount)

	conn.Broadcast(neffos.Message{
		Namespace: msg.Namespace,
		Event:     "OnVisit",
		Body:      []byte(fmt.Sprintf("%d", newCount)),
	})

	return nil
}

func (c *websocketController) OnChat(conn *neffos.NSConn, msg neffos.Message) error {
	c.service.Log("[ID: %s] broadcast to other clients the message [%s]", conn, string(msg.Body))

	ctx := websocket.GetContext(conn.Conn)
	ctx.Application().Logger().Infof("[IP: %s] [ID: %s] broadcast to other clients the message [%s]", ctx.RemoteAddr(), conn, string(msg.Body))

	conn.BroadcastOthers(msg)
	return nil
}
```

In this example, we define a `websocketController` with methods for handling WebSocket events such as `OnNamespaceConnected`, `OnNamespaceDisconnect`, and `OnChat`. We register the `websocketController` with the Iris application using the `mvc.Websocket` function.

<details><summary>Javascript Code (./browser/index.html)</summary>

```html
<html>

<head>
    <title>Online visitors MVC example</title>
    <style>
        body {
            margin: 0;
            font-family: -apple-system, "San Francisco", "Helvetica Neue", "Noto", "Roboto", "Calibri Light", sans-serif;
            color: #212121;
            font-size: 1.0em;
            line-height: 1.6;
        }

        .container {
            max-width: 750px;
            margin: auto;
            padding: 15px;
        }

        #online_visitors {
            font-weight: bold;
            font-size: 18px;
        }
    </style>
</head>

<body>
    <div class="container">
        <span id="online_visitors">1 online visitor</span>
    </div>

    <!-- the message's input -->
    <input id="input" type="text" />

    <!-- when clicked then a websocket event will be sent to the server, at this example we registered the 'chat' -->
    <button id="sendBtn" disabled>Send</button>

    <!-- the messages will be shown here -->
    <pre id="output"></pre>
    <!-- import the iris client-side library for browser from a CDN or locally.
     However, `neffos.(min.)js` is a NPM package too so alternatively,
     you can use it as dependency on your package.json and all nodejs-npm tooling become available:
     see the "browserify" example for more-->
    <script src="https://cdn.jsdelivr.net/npm/neffos.js@latest/dist/neffos-es5.js"></script>
    <script type="text/javascript">
        const wsURL = "ws://localhost:8080/websocket"
        var outputTxt = document.getElementById("output");
        function addMessage(msg) {
            outputTxt.innerHTML += msg + "\n";
        }

        async function runExample() {
            try {
                const conn = await neffos.dial(wsURL, {
                    default: { // "default" namespace.
                        _OnNamespaceConnected: function (nsConn, msg) {
                            if (nsConn.conn.wasReconnected()) {
                                addMessage("re-connected after " + nsConn.conn.reconnectTries.toString() + " trie(s)");
                            }

                            let inputTxt = document.getElementById("input");
                            let sendBtn = document.getElementById("sendBtn");

                            sendBtn.disabled = false;
                            sendBtn.onclick = function () {
                                const input = inputTxt.value;
                                inputTxt.value = "";
                                nsConn.emit("OnChat", input);
                                addMessage("Me: " + input);
                            };

                            addMessage("connected to namespace: " + msg.Namespace);
                        },
                        _OnNamespaceDisconnect: function (nsConn, msg) {
                            addMessage("disconnected from namespace: " + msg.Namespace);
                        },
                        OnChat: function (nsConn, msg) { // "OnChat" event.
                            console.log(msg);

                            addMessage(msg.Body);
                        },
                        OnVisit: function (nsConn, msg) {
                            const newCount = Number(msg.Body); // or parseInt.
                            console.log("visit websocket event with newCount of: ", newCount);

                            var text = "1 online visitor";
                            if (newCount > 1) {
                                text = newCount + " online visitors";
                            }
                            document.getElementById("online_visitors").innerHTML = text;
                        },
                    }
                });

                conn.connect("default");
            } catch (err) {
                console.log(err)
            }
        }

        runExample();
    </script>

</body>

</html>
```

</details>

### Dependency Injection

Dependency injection is a technique where an object receives other objects that it depends on. In the Iris MVC framework, you can register dependencies that will be automatically injected into your controllers.

In the example above, we register a `testService` as a dependency:

```go
app.RegisterDependency(app.Logger, newTestService)
```

This registers the `app.Logger` and makes it available for injection into the next `testService` dependency. Then it registers the  `testService` and makes it available for injection into the `websocketController`.

### The `newWebsocketController` Function

The `newWebsocketController` function is a constructor function that creates and returns a new instance of the `websocketController`. It takes a `testService` as a parameter, which is injected by the Iris dependency injection system.

Here is the `newWebsocketController` function:

```go
func newWebsocketController(service *testService) *websocketController {
	return &websocketController{
		service: service,
		counter: counter.NewCounter[int](),
	}
}
```

In this function, we create a new `websocketController` and initialize its `service` field with the injected `testService` instance. We also initialize a `counter` to keep track of the number of connected clients.

### WebSocket Controller Methods

The `websocketController` defines several methods to handle WebSocket events:

- `Namespace() string`: Returns the namespace for the WebSocket events.
- `OnNamespaceConnected(conn *neffos.NSConn, msg neffos.Message) error`: Handles the event when a client connects to the namespace.
- `OnNamespaceDisconnect(conn *neffos.NSConn, msg neffos.Message) error`: Handles the event when a client disconnects from the namespace.
- `OnChat(conn *neffos.NSConn, msg neffos.Message) error`: Handles the event when a client sends a chat message.

These methods use the `testService` to log messages and the `counter` to keep track of the number of connected clients. They also use the `conn.Broadcast` method to send messages to other connected clients.
