# ![Logo created by @santoshanand](logo_white_35_24.png) Iris

Iris is a fast, simple and efficient micro web framework for Go. It provides a beautifully expressive and easy to use foundation for your next website, API, or distributed app.

### About our User Experience Report

A week ago, _at 03 October_, we announced the first [Iris User Experience form-based Report](https://docs.google.com/forms/d/e/1FAIpQLSdCxZXPANg_xHWil4kVAdhmh7EBBHQZ_4_xSZVDL-oCC_z5pA/viewform?usp=sf_link) to let us learn more about you and any issues that troubles you with Iris (if any).

At overall, the results (so far) are very promising, high number of participations and the answers to the questions are near to the green feedback we were receiving over the past months from Gophers worldwide via our [rocket chat](https://chat.iris-go.com) and [author's twitter](https://twitter.com/makismaropoulos). **If you didn't complete the form yet, [please do so](https://docs.google.com/forms/d/e/1FAIpQLSdCxZXPANg_xHWil4kVAdhmh7EBBHQZ_4_xSZVDL-oCC_z5pA/viewform?usp=sf_link) as soon as possible!**

However, as everything in life; nothing goes as expected, people are strange, we programmers even more. The last part of the form has a text area which participiations can add any "questions or comments", there we saw one comment that surprised me the most, in the bad sense. We respect all individual singularities the same, we do not discriminate between people. The data are anonymous, so the only place to answer to that person is, _surprisingly_, here!

<details>
<summary>"I admire your dedication to iris and I am in love with its speed..."</summary>

The comment was "I admire your dedication to iris and I am in love with its speed but.. I've read some things on that blog and blablabla..." you get the point, at the first we were happy and suddenly we saw that "but... I've" and we broke xD.

The answer to this is clear in simple words so that anyone can understand; Did you really believed those unsubstantial things even if you could take some time off to read the source code?🤔

Iris was one of the top github trending projects written in Go Programming Language for the 2016 and the most trending web framework in the globe. We couldn't even imagine that we will be the receivers of countless "[thank you for iris, finally a web framework I can work on](https://twitter.com/_mgale/status/818591490305761280)" comments from hundreds strangers around the globe!

Please do research before digestion, those blog posts are not always telling the whole truth, they are not so innocent :)

Especially those from that kid that even don't correspond to reality;

```go
/* start */
```

First of all, that article **is referring 1.5 years ago**, to pretend that this article speaks for the present is hilariously ridiculous! Iris is on version 8 now and it's not a router any more, it's a fully featured web framework with its own ecosystem.

1. Iris does NOT use any third-party code inside it, like "httprouter" or "fasthttp". Just navigate to the source code. If you care about historical things you can search the project but it doesn't matter because the internal implementation of Iris changed a lot of times, a lot more than its public API changes:P.
2. Iris makes use of its own routing mechanisms with a unique **language interpreter** in order to serve even the most demanding of us `/user/{id:int min(2)}`, `/alphabetical/{param:string regexp(^[a-zA-Z ]+$)}` et cetera.
3. Iris has its own unique MVC architectural parser with heart-breaking performance.
4. Was it possible to do all those things and [much more](_examples) before Iris? Exactly. Iris offers you all these for free, plus the unmatched performance.
5. Iris is the result of hundreds(or thousands(?)) of hours of **FREE and UNPAID** work. There are people who actually found a decent job because of Iris. Thousands of Gophers are watching or/and helping to make Iris even better, the silent majority loves Iris even more.

That 23 years old, inhibited boy, who published that post had played you with the most immoral way! Reading the Iris' source code doesn't cost you a thing! Iris is free to use for everyone, Iris is an open-source software, no hidden spots. **Don't stuck in the past, get over that, Iris has succeed, move on now.**

```go
/* end */
```

</details>

_Psst_, we've produced a small video about your feelings regrating to Iris! You can watch the whole video at https://www.youtube.com/watch?v=jGx0LkuUs4A.

[![build status](https://img.shields.io/travis/kataras/iris/master.svg?style=flat-square)](https://travis-ci.org/kataras/iris)
[![report card](https://img.shields.io/badge/report%20card-a%2B-ff3333.svg?style=flat-square)](http://goreportcard.com/report/kataras/iris)
[![github issues](https://img.shields.io/github/issues/kataras/iris.svg?style=flat-square)](https://github.com/kataras/iris/issues?q=is%3Aopen+is%3Aissue)
[![github closed issues](https://img.shields.io/github/issues-closed-raw/kataras/iris.svg?style=flat-square)](https://github.com/kataras/iris/issues?q=is%3Aissue+is%3Aclosed)
[![release](https://img.shields.io/github/release/kataras/iris.svg?style=flat-square)](https://github.com/kataras/iris/releases)
[![view examples](https://img.shields.io/badge/learn%20by-examples-0077b3.svg?style=flat-square)](https://github.com/kataras/iris/tree/master/_examples)
[![chat](https://img.shields.io/badge/community-%20chat-00BCD4.svg?style=flat-square)](https://kataras.rocket.chat/channel/iris)

[![Iris vs .NET Core(C#) vs Node.js (Express)](https://iris-go.com/images/benchmark-new-gray.png)](_benchmarks)

</p>

## Installation

The only requirement is the [Go Programming Language](https://golang.org/dl/), at least version 1.9

```sh
$ go get -u github.com/kataras/iris
```

- Iris takes advantage of the [vendor directory](https://docs.google.com/document/d/1Bz5-UB7g2uPBdOx-rw5t9MxJwkfpx90cqG9AFL0JAYo) feature. You get truly reproducible builds, as this method guards against upstream renames and deletes.

- [Latest changes | v8.5.1](https://github.com/kataras/iris/blob/master/HISTORY.md#tu-10-october-2017--v851)

## Getting Started

```go
package main

import "github.com/kataras/iris"

func main() {
    app := iris.New()
    // Load all templates from the "./views" folder
    // where extension is ".html" and parse them
    // using the standard `html/template` package.
    app.RegisterView(iris.HTML("./views", ".html"))

    // Method:    GET
    // Resource:  http://localhost:8080
    app.Get("/", func(ctx iris.Context) {
        // Bind: {{.message}} with "Hello world!"
        ctx.ViewData("message", "Hello world!")
        // Render template file: ./views/hello.html
        ctx.View("hello.html")
    })

    // Method:    GET
    // Resource:  http://localhost:8080/user/42
    //
    // Need to use a custom regexp instead?
    // Easy;
    // Just mark the parameter's type to 'string'
    // which accepts anything and make use of
    // its `regexp` macro function, i.e:
    // app.Get("/user/{id:string regexp(^[0-9]+$)}")
    app.Get("/user/{id:long}", func(ctx iris.Context) {
        userID, _ := ctx.Params().GetInt64("id")
        ctx.Writef("User ID: %d", userID)
    })

    // Start the server using a network address.
    app.Run(iris.Addr(":8080"))
}
```

> Learn more about path parameter's types by clicking [here](https://github.com/kataras/iris/blob/master/_examples/routing/dynamic-path/main.go#L31).

```html
<!-- file: ./views/hello.html -->
<html>
<head>
    <title>Hello Page</title>
</head>
<body>
    <h1>{{.message}}</h1>
</body>
</html>
```

```sh
$ go run main.go
> Now listening on: http://localhost:8080
> Application started. Press CTRL+C to shut down.
```

Guidelines for bootstrapping handler-based applications can be found at the [_examples/structuring/handler-based](_examples/structuring/handler-based) folder.

### Quick MVC Tutorial

```go
package main

import (
    "github.com/kataras/iris"
    "github.com/kataras/iris/mvc"
)

func main() {
    app := iris.New()

    app.Controller("/helloworld", new(HelloWorldController))

    app.Run(iris.Addr("localhost:8080"))
}

type HelloWorldController struct {
    mvc.Controller

    // [ Your fields here ]
    // Request lifecycle data
    // Models
    // Database
    // Global properties
}

//
// GET: /helloworld

func (c *HelloWorldController) Get() string {
    return "This is my default action..."
}

//
// GET: /helloworld/{name:string}

func (c *HelloWorldController) GetBy(name string) string {
    return "Hello " + name
}

//
// GET: /helloworld/welcome

func (c *HelloWorldController) GetWelcome() (string, int) {
    return "This is the GetWelcome action func...", iris.StatusOK
}

//
// GET: /helloworld/welcome/{name:string}/{numTimes:int}

func (c *HelloWorldController) GetWelcomeBy(name string, numTimes int) {
    // Access to the low-level Context,
    // output arguments are optional of course so we don't have to use them here.
    c.Ctx.Writef("Hello %s, NumTimes is: %d", name, numTimes)
}
```

> The [_examples/mvc](_examples/mvc) and [mvc/controller_test.go](https://github.com/kataras/iris/blob/master/mvc/controller_test.go) files explain each feature with simple paradigms, they show how you can take advandage of the Iris MVC Binder, Iris MVC Models and many more...

Every `exported` func prefixed with an HTTP Method(`Get`, `Post`, `Put`, `Delete`...) in a controller is callable as an HTTP endpoint. In the sample above, all funcs writes a string to the response. Note the comments preceding each method.

An HTTP endpoint is a targetable URL in the web application, such as `http://localhost:8080/helloworld`, and combines the protocol used: HTTP, the network location of the web server (including the TCP port): `localhost:8080` and the target URI `/helloworld`.

The first comment states this is an [HTTP GET](https://www.w3schools.com/tags/ref_httpmethods.asp) method that is invoked by appending "/helloworld" to the base URL. The third comment specifies an [HTTP GET](https://www.w3schools.com/tags/ref_httpmethods.asp) method that is invoked by appending "/helloworld/welcome" to the URL.

Controller knows how to handle the "name" on `GetBy` or the "name" and "numTimes" at `GetWelcomeBy`, because of the `By` keyword, and builds the dynamic route without boilerplate; the third comment specifies an [HTTP GET](https://www.w3schools.com/tags/ref_httpmethods.asp) dynamic method that is invoked by any URL that starts with "/helloworld/welcome" and followed by two more path parts, the first one can accept any value and the second can accept only numbers, i,e: "http://localhost:8080/helloworld/welcome/golang/32719", otherwise a [404 Not Found HTTP Error](https://www.w3.org/Protocols/rfc2616/rfc2616-sec10.html#sec10.4.5) will be sent to the client instead.

### Quick MVC Tutorial #2

Iris has a very powerful and **blazing [fast](_benchmarks)** MVC support, you can return any value of any type from a method function
and it will be sent to the client as expected.

* if `string` then it's the body.
* if `string` is the second output argument then it's the content type.
* if `int` then it's the status code.
* if `error` and not nil then (any type) response will be omitted and error's text with a 400 bad request will be rendered instead.
* if `(int, error)` and error is not nil then the response result will be the error's text with the status code as `int`.
* if  `custom struct` or `interface{}` or `slice` or `map` then it will be rendered as json, unless a `string` content type is following.
* if `mvc.Result` then it executes its `Dispatch` function, so good design patters can be used to split the model's logic where needed.

The example below is not intended to be used in production but it's a good showcase of some of the return types we saw before;

```go
package main

import (
    "github.com/kataras/iris"
    "github.com/kataras/iris/middleware/basicauth"
    "github.com/kataras/iris/mvc"
)

// Movie is our sample data structure.
type Movie struct {
    Name   string `json:"name"`
    Year   int    `json:"year"`
    Genre  string `json:"genre"`
    Poster string `json:"poster"`
}

// movies contains our imaginary data source.
var movies = []Movie{
    {
        Name:   "Casablanca",
        Year:   1942,
        Genre:  "Romance",
        Poster: "https://iris-go.com/images/examples/mvc-movies/1.jpg",
    },
    {
        Name:   "Gone with the Wind",
        Year:   1939,
        Genre:  "Romance",
        Poster: "https://iris-go.com/images/examples/mvc-movies/2.jpg",
    },
    {
        Name:   "Citizen Kane",
        Year:   1941,
        Genre:  "Mystery",
        Poster: "https://iris-go.com/images/examples/mvc-movies/3.jpg",
    },
    {
        Name:   "The Wizard of Oz",
        Year:   1939,
        Genre:  "Fantasy",
        Poster: "https://iris-go.com/images/examples/mvc-movies/4.jpg",
    },
}


var basicAuth = basicauth.New(basicauth.Config{
    Users: map[string]string{
        "admin": "password",
    },
})


func main() {
    app := iris.New()

    app.Use(basicAuth)

    app.Controller("/movies", new(MoviesController))

    app.Run(iris.Addr(":8080"))
}

// MoviesController is our /movies controller.
type MoviesController struct {
    // mvc.C is just a lightweight alternative
    // to the "mvc.Controller" controller type.
    mvc.C
}

// Get returns list of the movies
// Demo:
// curl -i http://localhost:8080/movies
func (c *MoviesController) Get() []Movie {
    return movies
}

// GetBy returns a movie
// Demo:
// curl -i http://localhost:8080/movies/1
func (c *MoviesController) GetBy(id int) Movie {
    return movies[id]
}

// PutBy updates a movie
// Demo:
// curl -i -X PUT -F "genre=Thriller" -F "poster=@/Users/kataras/Downloads/out.gif" http://localhost:8080/movies/1
func (c *MoviesController) PutBy(id int) Movie {
    // get the movie
    m := movies[id]

    // get the request data for poster and genre
    file, info, err := c.Ctx.FormFile("poster")
    if err != nil {
        c.Ctx.StatusCode(iris.StatusInternalServerError)
        return Movie{}
    }
    file.Close()            // we don't need the file
    poster := info.Filename // imagine that as the url of the uploaded file...
    genre := c.Ctx.FormValue("genre")

    // update the poster
    m.Poster = poster
    m.Genre = genre
    movies[id] = m

    return m
}

// DeleteBy deletes a movie
// Demo:
// curl -i -X DELETE -u admin:password http://localhost:8080/movies/1
func (c *MoviesController) DeleteBy(id int) iris.Map {
    // delete the entry from the movies slice
    deleted := movies[id].Name
    movies = append(movies[:id], movies[id+1:]...)
    // and return the deleted movie's name
    return iris.Map{"deleted": deleted}
}
```

### Quick MVC Tutorial #3

Nothing stops you from using your favorite **folder structure**. Iris is a low level web framework, it has got MVC first-class support but it doesn't limit your folder structure, this is your choice.

Structuring depends on your own needs. We can't tell you how to design your own application for sure but you're free to take a closer look to one typical example below;

[![folder structure example](_examples/mvc/using-method-result/folder_structure.png)](_examples/mvc/using-method-result)

Shhh, let's spread the code itself.

```go
// file: controllers/hello_controller.go

package controllers

import (
    "errors"

    "github.com/kataras/iris/mvc"
)

// HelloController is our sample controller
// it handles GET: /hello and GET: /hello/{name}
type HelloController struct {
    mvc.C
}

var helloView = mvc.View{
    Name: "hello/index.html",
    Data: map[string]interface{}{
        "Title":     "Hello Page",
        "MyMessage": "Welcome to my awesome website",
    },
}

// Get will return a predefined view with bind data.
//
// `mvc.Result` is just an interface with a `Dispatch` function.
// `mvc.Response` and `mvc.View` are the built'n result type dispatchers
// you can even create custom response dispatchers by
// implementing the `github.com/kataras/iris/mvc#Result` interface.
func (c *HelloController) Get() mvc.Result {
    return helloView
}

// you can define a standard error in order to be re-usable anywhere in your app.
var errBadName = errors.New("bad name")

// you can just return it as error or even better
// wrap this error with an mvc.Response to make it an mvc.Result compatible type.
var badName = mvc.Response{Err: errBadName, Code: 400}

// GetBy returns a "Hello {name}" response.
// Demos:
// curl -i http://localhost:8080/hello/iris
// curl -i http://localhost:8080/hello/anything
func (c *HelloController) GetBy(name string) mvc.Result {
    if name != "iris" {
        return badName
        // or
        // GetBy(name string) (mvc.Result, error) {
        //  return nil, errBadName
        // }
    }

    // return mvc.Response{Text: "Hello " + name} OR:
    return mvc.View{
        Name: "hello/name.html",
        Data: name,
    }
}
```

```html
<!-- file: views/hello/index.html -->
<html>

<head>
    <title>{{.Title}} - My App</title>
</head>

<body>
    <p>{{.MyMessage}}</p>
</body>

</html>
```

```html
<!-- file: views/hello/name.html -->
<html>

<head>
    <title>{{.}}' Portfolio - My App</title>
</head>

<body>
    <h1>Hello {{.}}</h1>
</body>

</html>
```

> Navigate to the [_examples/view](_examples/#view) for more examples
like shared layouts, tmpl funcs, reverse routing and more!

```go
// file: models/movie.go

package models

import "github.com/kataras/iris/context"

// Movie is our sample data structure.
type Movie struct {
    ID     int64  `json:"id"`
    Name   string `json:"name"`
    Year   int    `json:"year"`
    Genre  string `json:"genre"`
    Poster string `json:"poster"`
}

// Dispatch completes the `kataras/iris/mvc#Result` interface.
// Sends a `Movie` as a controlled http response.
// If its ID is zero or less then it returns a 404 not found error
// else it returns its json representation,
// (just like the controller's functions do for custom types by default).
//
// Don't overdo it, the application's logic should not be here.
// It's just one more step of validation before the response,
// simple checks can be added here.
//
// It's just a showcase,
// imagine the potentials this feature gives when designing a bigger application.
//
// This is called where the return value from a controller's method functions
// is type of `Movie`.
// For example the `controllers/movie_controller.go#GetBy`.
func (m Movie) Dispatch(ctx context.Context) {
    if m.ID <= 0 {
        ctx.NotFound()
        return
    }
    ctx.JSON(m, context.JSON{Indent: " "})
}
```

> For those who wonder `iris.Context`(go 1.9 type alias feature) and `context.Context` is the same [exact thing](faq.md#type-aliases).

```go
// file: services/movie_service.go

package services

import (
    "errors"
    "sync"

    "github.com/kataras/iris/_examples/mvc/using-method-result/models"
)

// MovieService handles CRUID operations of a movie entity/model.
// It's here to decouple the data source from the higher level compoments.
// As a result a different service for a specific datasource (or repository)
// can be used from the main application without any additional changes.
type MovieService interface {
    GetSingle(query func(models.Movie) bool) (movie models.Movie, found bool)
    GetByID(id int64) (models.Movie, bool)

    InsertOrUpdate(movie models.Movie) (models.Movie, error)
    DeleteByID(id int64) bool

    GetMany(query func(models.Movie) bool, limit int) (result []models.Movie)
    GetAll() []models.Movie
}

// NewMovieServiceFromMemory returns a new memory-based movie service.
func NewMovieServiceFromMemory(source map[int64]models.Movie) MovieService {
    return &MovieMemoryService{
        source: source,
    }
}

/*
A Movie Service can have different data sources:
func NewMovieServiceFromDB(db datasource.MySQL) {
    return &MovieDatabaseService{
        db: db,
    }
}

Another pattern is to initialize the database connection
or any source here based on a "string" name or an "enum".
func NewMovieService(source string) MovieService {
    if source == "memory" {
        return NewMovieServiceFromMemory(datasource.Movies)
    }
    if source == "database" {
        db = datasource.NewDB("....")
        return NewMovieServiceFromDB(db)
    }
    [...]
    return nil
}
*/

// MovieMemoryService is a "MovieService"
// which manages the movies using the memory data source (map).
type MovieMemoryService struct {
    source map[int64]models.Movie
    mu     sync.RWMutex
}

// GetSingle receives a query function
// which is fired for every single movie model inside
// our imaginary data source.
// When that function returns true then it stops the iteration.
//
// It returns the query's return last known boolean value
// and the last known movie model
// to help callers to reduce the LOC.
//
// It's actually a simple but very clever prototype function
// I'm using everywhere since I firstly think of it,
// hope you'll find it very useful as well.
func (s *MovieMemoryService) GetSingle(query func(models.Movie) bool) (movie models.Movie, found bool) {
    s.mu.RLock()
    for _, movie = range s.source {
        found = query(movie)
        if found {
            break
        }
    }
    s.mu.RUnlock()

    // set an empty models.Movie if not found at all.
    if !found {
        movie = models.Movie{}
    }

    return
}

// GetByID returns a movie based on its id.
// Returns true if found, otherwise false, the bool should be always checked
// because the models.Movie may be filled with the latest element
// but not the correct one, although it can be used for debugging.
func (s *MovieMemoryService) GetByID(id int64) (models.Movie, bool) {
    return s.GetSingle(func(m models.Movie) bool {
        return m.ID == id
    })
}

// InsertOrUpdate adds or updates a movie to the (memory) storage.
//
// Returns the new movie and an error if any.
func (s *MovieMemoryService) InsertOrUpdate(movie models.Movie) (models.Movie, error) {
    id := movie.ID

    if id == 0 { // Create new action
        var lastID int64
        // find the biggest ID in order to not have duplications
        // in productions apps you can use a third-party
        // library to generate a UUID as string.
        s.mu.RLock()
        for _, item := range s.source {
            if item.ID > lastID {
                lastID = item.ID
            }
        }
        s.mu.RUnlock()

        id = lastID + 1
        movie.ID = id

        // map-specific thing
        s.mu.Lock()
        s.source[id] = movie
        s.mu.Unlock()

        return movie, nil
    }

    // Update action based on the movie.ID,
    // here we will allow updating the poster and genre if not empty.
    // Alternatively we could do pure replace instead:
    // s.source[id] = movie
    // and comment the code below;
    current, exists := s.GetByID(id)
    if !exists { // ID is not a real one, return an error.
        return models.Movie{}, errors.New("failed to update a nonexistent movie")
    }

    // or comment these and s.source[id] = m for pure replace
    if movie.Poster != "" {
        current.Poster = movie.Poster
    }

    if movie.Genre != "" {
        current.Genre = movie.Genre
    }

    // map-specific thing
    s.mu.Lock()
    s.source[id] = current
    s.mu.Unlock()

    return movie, nil
}

// DeleteByID deletes a movie by its id.
//
// Returns true if deleted otherwise false.
func (s *MovieMemoryService) DeleteByID(id int64) bool {
    if _, exists := s.GetByID(id); !exists {
        // we could do _, exists := s.source[id] instead
        // but we don't because you should learn
        // how you can use that service's functions
        // with any other source, i.e database.
        return false
    }

    // map-specific thing
    s.mu.Lock()
    delete(s.source, id)
    s.mu.Unlock()

    return true
}

// GetMany same as GetSingle but returns one or more models.Movie as a slice.
// If limit <=0 then it returns everything.
func (s *MovieMemoryService) GetMany(query func(models.Movie) bool, limit int) (result []models.Movie) {
    loops := 0

    s.mu.RLock()
    for _, movie := range s.source {
        loops++

        passed := query(movie)
        if passed {
            result = append(result, movie)
        }
        // we have to return at least one movie if "passed" was true.
        if limit >= loops {
            break
        }
    }
    s.mu.RUnlock()

    return
}

// GetAll returns all movies.
func (s *MovieMemoryService) GetAll() []models.Movie {
    movies := s.GetMany(func(m models.Movie) bool { return true }, -1)
    return movies
}
```

```go
// file: controllers/movie_controller.go

package controllers

import (
    "errors"

    "github.com/kataras/iris/_examples/mvc/using-method-result/models"
    "github.com/kataras/iris/_examples/mvc/using-method-result/services"

    "github.com/kataras/iris"
    "github.com/kataras/iris/mvc"
)

// MovieController is our /movies controller.
type MovieController struct {
    // mvc.C is just a lightweight lightweight alternative
    // to the "mvc.Controller" controller type,
    // use it when you don't need mvc.Controller's fields
    // (you don't need those fields when you return values from the method functions).
    mvc.C

    // Our MovieService, it's an interface which
    // is binded from the main application.
    Service services.MovieService
}

// Get returns list of the movies.
// Demo:
// curl -i http://localhost:8080/movies
func (c *MovieController) Get() []models.Movie {
    return c.Service.GetAll()
}

// GetBy returns a movie.
// Demo:
// curl -i http://localhost:8080/movies/1
func (c *MovieController) GetBy(id int64) models.Movie {
    m, _ := c.Service.GetByID(id)
    return m
}

// PutBy updates a movie.
// Demo:
// curl -i -X PUT -F "genre=Thriller" -F "poster=@/Users/kataras/Downloads/out.gif" http://localhost:8080/movies/1
func (c *MovieController) PutBy(id int64) (models.Movie, error) {
    // get the request data for poster and genre
    file, info, err := c.Ctx.FormFile("poster")
    if err != nil {
        return models.Movie{}, errors.New("failed due form file 'poster' missing")
    }
    // we don't need the file so close it now.
    file.Close()

    // imagine that is the url of the uploaded file...
    poster := info.Filename
    genre := c.Ctx.FormValue("genre")

    // update the movie and return it.
    return c.Service.InsertOrUpdate(models.Movie{
        ID:     id,
        Poster: poster,
        Genre:  genre,
    })
}

// DeleteBy deletes a movie.
// Demo:
// curl -i -X DELETE -u admin:password http://localhost:8080/movies/1
func (c *MovieController) DeleteBy(id int64) interface{} {
    wasDel := c.Service.DeleteByID(id)
    if wasDel {
        // and return the deleted movie's ID
        return iris.Map{"deleted": id}
    }
    // here we can see that a method function can return any of those two types(map or int),
    // we don't have to specify the return type to a specific type.
    return iris.StatusBadRequest
}
```

```go
// file: datasource/movies.go

package datasource

import "github.com/kataras/iris/_examples/mvc/using-method-result/models"

// Movies is our imaginary data source.
var Movies = map[int64]models.Movie{
    1: {
        ID:     1,
        Name:   "Casablanca",
        Year:   1942,
        Genre:  "Romance",
        Poster: "https://iris-go.com/images/examples/mvc-movies/1.jpg",
    },
    2: {
        ID:     2,
        Name:   "Gone with the Wind",
        Year:   1939,
        Genre:  "Romance",
        Poster: "https://iris-go.com/images/examples/mvc-movies/2.jpg",
    },
    3: {
        ID:     3,
        Name:   "Citizen Kane",
        Year:   1941,
        Genre:  "Mystery",
        Poster: "https://iris-go.com/images/examples/mvc-movies/3.jpg",
    },
    4: {
        ID:     4,
        Name:   "The Wizard of Oz",
        Year:   1939,
        Genre:  "Fantasy",
        Poster: "https://iris-go.com/images/examples/mvc-movies/4.jpg",
    },
    5: {
        ID:     5,
        Name:   "North by Northwest",
        Year:   1959,
        Genre:  "Thriller",
        Poster: "https://iris-go.com/images/examples/mvc-movies/5.jpg",
    },
}
```

```go
// file: middleware/basicauth.go

package middleware

import "github.com/kataras/iris/middleware/basicauth"

// BasicAuth middleware sample.
var BasicAuth = basicauth.New(basicauth.Config{
    Users: map[string]string{
        "admin": "password",
    },
})
```

```go
// file: main.go

package main

import (
    "github.com/kataras/iris/_examples/mvc/using-method-result/controllers"
    "github.com/kataras/iris/_examples/mvc/using-method-result/datasource"
    "github.com/kataras/iris/_examples/mvc/using-method-result/middleware"
    "github.com/kataras/iris/_examples/mvc/using-method-result/services"

    "github.com/kataras/iris"
)

func main() {
    app := iris.New()

    // Load the template files.
    app.RegisterView(iris.HTML("./views", ".html"))

    // Register our controllers.
    app.Controller("/hello", new(controllers.HelloController))

    // Create our movie service (memory), we will bind it to the movies controller.
    service := services.NewMovieServiceFromMemory(datasource.Movies)

    app.Controller("/movies", new(controllers.MovieController),
        // Bind the "service" to the MovieController's Service (interface) field.
        service,
        // Add the basic authentication(admin:password) middleware
        // for the /movies based requests.
        middleware.BasicAuth)

    // Start the web server at localhost:8080
    // http://localhost:8080/hello
    // http://localhost:8080/hello/iris
    // http://localhost:8080/movies/1
    app.Run(
        iris.Addr("localhost:8080"),
        iris.WithoutVersionChecker,
        iris.WithoutServerError(iris.ErrServerClosed),
        iris.WithOptimizations, // enables faster json serialization and more
    )
}
```

More folder structure guidelines can be found at the [_examples/#structuring](_examples/#structuring) section.

## 😃 Do you like what you see so far?

> Prepare yourself a cup of coffee, or tea, whatever enjoys you the most!

- [How to build a file upload form using DropzoneJS and Go](https://hackernoon.com/how-to-build-a-file-upload-form-using-dropzonejs-and-go-8fb9f258a991)
- [How to display existing files on server using DropzoneJS and Go](https://hackernoon.com/how-to-display-existing-files-on-server-using-dropzonejs-and-go-53e24b57ba19)
- [Iris Go vs .NET Core Kestrel in terms of HTTP performance](https://hackernoon.com/iris-go-vs-net-core-kestrel-in-terms-of-http-performance-806195dc93d5)
- [Go vs .NET Core in terms of HTTP performance](https://medium.com/@kataras/go-vs-net-core-in-terms-of-http-performance-7535a61b67b8)
- [Iris, a modular web framework](https://medium.com/@corebreaker/iris-web-cd684b4685c7)
- [Deploying a Iris Golang app in hasura](https://medium.com/@HasuraHQ/deploy-an-iris-golang-app-with-backend-apis-in-minutes-25a559bf530b)
- [How to Turn an Android Device into a Web Server](https://twitter.com/ThePracticalDev/status/892022594031017988)
- [A URL Shortener Service using Go, Iris and Bolt](https://medium.com/@kataras/a-url-shortener-service-using-go-iris-and-bolt-4182f0b00ae7)
- [Why I preferred Go over Node.js for simple Web Application](https://medium.com/@tigranbs/why-i-preferred-go-over-node-js-for-simple-web-application-d4a549e979b9)

Take some time, `don't say we didn't warn you`,  and continue your journey by [navigating to the next README page](README_NEXT.md).

## License

Iris is licensed under the 3-Clause BSD [License](LICENSE). Iris is 100% open-source software.