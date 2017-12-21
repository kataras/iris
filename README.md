## I'm working hard on the [dev](https://github.com/kataras/iris/tree/dev) branch for the next release of Iris. 

Do you remember, last Christmas? I did publish the version 6 with net/http and HTTP/2 support, and you've embraced Iris with so much love, ultimately it was a successful move.

I tend to make surprises by giving you the most unique and useful features, especially on Christmas period.

This year, I intend to give you more gifts.

Don't worry, it will not contain any breaking changes, except of some MVC concepts that are re-designed.

The new Iris' MVC Ecosystem is ready on the [dev/mvc](https://github.com/kataras/iris/tree/dev/mvc). It contains features that you've never saw before, in any programming language framework. It is also, by far, the fastest MVC implementation ever created, very close to raw handlers - it's Iris, it's superior, we couldn't expect something different after all :) Treat that with respect as it treats you :)

I'm doing my bests to get it ready before Christmas.

Star or watch the repository to stay up to date and get ready for the most amazing features!

Yours faithfully, [Gerasimos Maropoulos](https://twitter.com/MakisMaropoulos).

------

# [![Logo created by @santoshanand](logo_white_35_24.png)](https://iris-go.com) Iris

[![build status](https://img.shields.io/travis/kataras/iris/master.svg?style=flat-square)](https://travis-ci.org/kataras/iris)[![report card](https://img.shields.io/badge/report%20card-a%2B-ff3333.svg?style=flat-square)](http://goreportcard.com/report/kataras/iris)[![github closed issues](https://img.shields.io/github/issues-closed-raw/kataras/iris.svg?style=flat-square)](https://github.com/kataras/iris/issues?q=is%3Aissue+is%3Aclosed)[![release](https://img.shields.io/github/release/kataras/iris.svg?style=flat-square)](https://github.com/kataras/iris/releases)[![view examples](https://img.shields.io/badge/learn%20by-examples-0077b3.svg?style=flat-square)](https://github.com/kataras/iris/tree/master/_examples)[![chat](https://img.shields.io/badge/community-%20chat-00BCD4.svg?style=flat-square)](https://kataras.rocket.chat/channel/iris)[![CLA assistant](https://cla-assistant.io/readme/badge/kataras/iris?style=flat-square)](https://cla-assistant.io/kataras/iris)

Iris is a fast, simple and efficient web framework for Go.

Iris provides a beautifully expressive and easy to use foundation for your next website, API, or distributed app.

Learn what [others say about Iris](https://www.youtube.com/watch?v=jGx0LkuUs4A) and [star](https://github.com/kataras/iris/stargazers) this github repository to stay [up to date](https://facebook.com/iris.framework).

[![Iris vs .NET Core(C#) vs Node.js (Express)](https://iris-go.com/images/benchmark-new.png)](_benchmarks/README_UNIX.md)

<details>
<summary>Benchmarks from third-party source over the rest web frameworks</summary>

![Comparison with other frameworks](https://raw.githubusercontent.com/smallnest/go-web-framework-benchmark/4db507a22c964c9bc9774c5b31afdc199a0fe8b7/benchmark.png)

_Updated at: [Tuesday, 21 November 2017](_benchmarks/README_UNIX.md)_
</details>

## Built with ♥️

We have no doubt you will able to find other web frameworks written in Go
and even put up a real fight to learn and use them for quite some time but
make no mistake, sooner or later you will be using Iris, not because of the ergonomic, high-performant solution that it provides but its well-documented unique features, as these will transform you to a real rockstar geek.

No matter what you're trying to build, Iris covers 
every type of application, from micro services to large monolithic web applications.
It's actually the best piece of software for back-end web developers
you can find online.

Iris may have reached version 8, but we're not stopping there. We have many feature ideas on our board that we're anxious to add and other innovative web development solutions that we're planning to build into Iris.

Accelerated by [KeyCDN](https://www.keycdn.com/), a simple, fast and reliable CDN.

We are developing this project using the best code editor for Golang; [Visual Studio Code](https://code.visualstudio.com/) supported by [Microsoft](https://www.microsoft.com).

If you're coming from [nodejs](https://nodejs.org) world, Iris is the [expressjs](https://github.com/expressjs/express) equivalent for Gophers.

## Table Of Content

* [Installation](#installation)
* [Latest changes](https://github.com/kataras/iris/blob/master/HISTORY.md#th-09-november-2017--v858)
* [Getting started](#getting-started)
* [Learn](_examples/)
    * [MVC (Model View Controller)](_examples/#mvc) **NEW**
    * [Structuring](_examples/#structuring) **NEW**
    * [HTTP Listening](_examples/#http-listening)
    * [Configuration](_examples/#configuration)
    * [Routing, Grouping, Dynamic Path Parameters, "Macros" and Custom Context](_examples/#routing-grouping-dynamic-path-parameters-macros-and-custom-context)
    * [Subdomains](_examples/#subdomains)
    * [Wrap `http.Handler/HandlerFunc`](_examples/#convert-httphandlerhandlerfunc)
    * [View](_examples/#view)
    * [Authentication](_examples/#authentication)
    * [File Server](_examples/#file-server)
    * [How to Read from `context.Request() *http.Request`](_examples/#how-to-read-from-contextrequest-httprequest)
    * [How to Write to `context.ResponseWriter() http.ResponseWriter`](_examples/#how-to-write-to-contextresponsewriter-httpresponsewriter)
    * [Test](_examples/#testing)	
    * [Cache](_examples/#caching)
    * [Sessions](_examples/#sessions)
    * [Websockets](_examples/#websockets)
    * [Miscellaneous](_examples/#miscellaneous)
    * [POC: Convert the medium-sized project "Parrot" from native to Iris](https://github.com/iris-contrib/parrot)
    * [POC: Isomorphic react/hot reloadable/redux/css-modules starter kit](https://github.com/kataras/iris-starter-kit)
    * [Typescript Automation Tools](typescript/#table-of-contents)
    * [Tutorial: A URL Shortener Service using Go, Iris and Bolt](https://medium.com/@kataras/a-url-shortener-service-using-go-iris-and-bolt-4182f0b00ae7)
    * [Tutorial: Online Visitors](_examples/tutorial/online-visitors)
    * [Tutorial: Caddy](_examples/tutorial/caddy)
    * [Tutorial: DropzoneJS Uploader](https://hackernoon.com/how-to-build-a-file-upload-form-using-dropzonejs-and-go-8fb9f258a991)
    * [Tutorial:Iris Go Framework + MongoDB](https://medium.com/go-language/iris-go-framework-mongodb-552e349eab9c)
* [Middleware](middleware/)
* [Dockerize](https://github.com/iris-contrib/cloud-native-go)
* [Contributing](CONTRIBUTING.md)
* [FAQ](FAQ.md)
* [What's next?](#now-you-are-ready-to-move-to-the-next-step-and-get-closer-to-becoming-a-pro-gopher)
* [People](#people)

## Installation

The only requirement is the [Go Programming Language](https://golang.org/dl/)

```sh
$ go get -u github.com/kataras/iris
```

Iris takes advantage of the [vendor directory](https://docs.google.com/document/d/1Bz5-UB7g2uPBdOx-rw5t9MxJwkfpx90cqG9AFL0JAYo) feature. You get truly reproducible builds, as this method guards against upstream renames and deletes.

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

![overview screen](https://github.com/kataras/build-a-better-web-together/raw/master/overview_screen_1.png)

> Wanna re-start your app automatically when source code changes happens? Install the [rizla](https://github.com/kataras/rizla) tool and run `rizla main.go` instead of `go run main.go`.

Guidelines for bootstrapping applications can be found at the [_examples/structuring](_examples/#structuring).

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
    mvc.C

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
* if `bool` is false then it throws 404 not found http error by skipping everything else.
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

[![folder structure example](_examples/mvc/overview/folder_structure.png)](_examples/mvc/overview)

Shhh, let's spread the code itself.

#### Data Model Layer

```go
// file: datamodels/movie.go

package datamodels

// Movie is our sample data structure.
// Keep note that the tags for public-use (for our web app)
// should be kept in other file like "web/viewmodels/movie.go"
// which could wrap by embedding the datamodels.Movie or
// declare new fields instead butwe will use this datamodel
// as the only one Movie model in our application,
// for the shake of simplicty.
type Movie struct {
    ID     int64  `json:"id"`
    Name   string `json:"name"`
    Year   int    `json:"year"`
    Genre  string `json:"genre"`
    Poster string `json:"poster"`
}
```

#### Data Source / Data Store Layer

```go
// file: datasource/movies.go

package datasource

import "github.com/kataras/iris/_examples/mvc/overview/datamodels"

// Movies is our imaginary data source.
var Movies = map[int64]datamodels.Movie{
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

#### Repositories

The layer which has direct access to the "datasource" and can manipulate data directly.

```go
// file: repositories/movie_repository.go

package repositories

import (
    "errors"
    "sync"

    "github.com/kataras/iris/_examples/mvc/overview/datamodels"
)

// Query represents the visitor and action queries.
type Query func(datamodels.Movie) bool

// MovieRepository handles the basic operations of a movie entity/model.
// It's an interface in order to be testable, i.e a memory movie repository or
// a connected to an sql database.
type MovieRepository interface {
    Exec(query Query, action Query, limit int, mode int) (ok bool)

    Select(query Query) (movie datamodels.Movie, found bool)
    SelectMany(query Query, limit int) (results []datamodels.Movie)

    InsertOrUpdate(movie datamodels.Movie) (updatedMovie datamodels.Movie, err error)
    Delete(query Query, limit int) (deleted bool)
}

// NewMovieRepository returns a new movie memory-based repository,
// the one and only repository type in our example.
func NewMovieRepository(source map[int64]datamodels.Movie) MovieRepository {
    return &movieMemoryRepository{source: source}
}

// movieMemoryRepository is a "MovieRepository"
// which manages the movies using the memory data source (map).
type movieMemoryRepository struct {
    source map[int64]datamodels.Movie
    mu     sync.RWMutex
}

const (
    // ReadOnlyMode will RLock(read) the data .
    ReadOnlyMode = iota
    // ReadWriteMode will Lock(read/write) the data.
    ReadWriteMode
)

func (r *movieMemoryRepository) Exec(query Query, action Query, actionLimit int, mode int) (ok bool) {
    loops := 0

    if mode == ReadOnlyMode {
        r.mu.RLock()
        defer r.mu.RUnlock()
    } else {
        r.mu.Lock()
        defer r.mu.Unlock()
    }

    for _, movie := range r.source {
        ok = query(movie)
        if ok {
            if action(movie) {
                loops++
                if actionLimit >= loops {
                    break // break
                }
            }
        }
    }

    return
}

// Select receives a query function
// which is fired for every single movie model inside
// our imaginary data source.
// When that function returns true then it stops the iteration.
//
// It returns the query's return last known "found" value
// and the last known movie model
// to help callers to reduce the LOC.
//
// It's actually a simple but very clever prototype function
// I'm using everywhere since I firstly think of it,
// hope you'll find it very useful as well.
func (r *movieMemoryRepository) Select(query Query) (movie datamodels.Movie, found bool) {
    found = r.Exec(query, func(m datamodels.Movie) bool {
        movie = m
        return true
    }, 1, ReadOnlyMode)

    // set an empty datamodels.Movie if not found at all.
    if !found {
        movie = datamodels.Movie{}
    }

    return
}

// SelectMany same as Select but returns one or more datamodels.Movie as a slice.
// If limit <=0 then it returns everything.
func (r *movieMemoryRepository) SelectMany(query Query, limit int) (results []datamodels.Movie) {
    r.Exec(query, func(m datamodels.Movie) bool {
        results = append(results, m)
        return true
    }, limit, ReadOnlyMode)

    return
}

// InsertOrUpdate adds or updates a movie to the (memory) storage.
//
// Returns the new movie and an error if any.
func (r *movieMemoryRepository) InsertOrUpdate(movie datamodels.Movie) (datamodels.Movie, error) {
    id := movie.ID

    if id == 0 { // Create new action
        var lastID int64
        // find the biggest ID in order to not have duplications
        // in productions apps you can use a third-party
        // library to generate a UUID as string.
        r.mu.RLock()
        for _, item := range r.source {
            if item.ID > lastID {
                lastID = item.ID
            }
        }
        r.mu.RUnlock()

        id = lastID + 1
        movie.ID = id

        // map-specific thing
        r.mu.Lock()
        r.source[id] = movie
        r.mu.Unlock()

        return movie, nil
    }

    // Update action based on the movie.ID,
    // here we will allow updating the poster and genre if not empty.
    // Alternatively we could do pure replace instead:
    // r.source[id] = movie
    // and comment the code below;
    current, exists := r.Select(func(m datamodels.Movie) bool {
        return m.ID == id
    })

    if !exists { // ID is not a real one, return an error.
        return datamodels.Movie{}, errors.New("failed to update a nonexistent movie")
    }

    // or comment these and r.source[id] = m for pure replace
    if movie.Poster != "" {
        current.Poster = movie.Poster
    }

    if movie.Genre != "" {
        current.Genre = movie.Genre
    }

    // map-specific thing
    r.mu.Lock()
    r.source[id] = current
    r.mu.Unlock()

    return movie, nil
}

func (r *movieMemoryRepository) Delete(query Query, limit int) bool {
    return r.Exec(query, func(m datamodels.Movie) bool {
        delete(r.source, m.ID)
        return true
    }, limit, ReadWriteMode)
}
```

#### Services

The layer which has access to call functions from the "repositories" and "models" (or even "datamodels" if simple application). It should contain the most of the domain logic.

```go
// file: services/movie_service.go

package services

import (
    "github.com/kataras/iris/_examples/mvc/overview/datamodels"
    "github.com/kataras/iris/_examples/mvc/overview/repositories"
)

// MovieService handles some of the CRUID operations of the movie datamodel.
// It depends on a movie repository for its actions.
// It's here to decouple the data source from the higher level compoments.
// As a result a different repository type can be used with the same logic without any aditional changes.
// It's an interface and it's used as interface everywhere
// because we may need to change or try an experimental different domain logic at the future.
type MovieService interface {
    GetAll() []datamodels.Movie
    GetByID(id int64) (datamodels.Movie, bool)
    DeleteByID(id int64) bool
    UpdatePosterAndGenreByID(id int64, poster string, genre string) (datamodels.Movie, error)
}

// NewMovieService returns the default movie service.
func NewMovieService(repo repositories.MovieRepository) MovieService {
    return &movieService{
        repo: repo,
    }
}

type movieService struct {
    repo repositories.MovieRepository
}

// GetAll returns all movies.
func (s *movieService) GetAll() []datamodels.Movie {
    return s.repo.SelectMany(func(_ datamodels.Movie) bool {
        return true
    }, -1)
}

// GetByID returns a movie based on its id.
func (s *movieService) GetByID(id int64) (datamodels.Movie, bool) {
    return s.repo.Select(func(m datamodels.Movie) bool {
        return m.ID == id
    })
}

// UpdatePosterAndGenreByID updates a movie's poster and genre.
func (s *movieService) UpdatePosterAndGenreByID(id int64, poster string, genre string) (datamodels.Movie, error) {
    // update the movie and return it.
    return s.repo.InsertOrUpdate(datamodels.Movie{
        ID:     id,
        Poster: poster,
        Genre:  genre,
    })
}

// DeleteByID deletes a movie by its id.
//
// Returns true if deleted otherwise false.
func (s *movieService) DeleteByID(id int64) bool {
    return s.repo.Delete(func(m datamodels.Movie) bool {
        return m.ID == id
    }, 1)
}
```

#### View Models

There should be the view models, the structure that the client will be able to see.

Example:

```go
import (
    "github.com/kataras/iris/_examples/mvc/overview/datamodels"

    "github.com/kataras/iris/context"
)

type Movie struct {
    datamodels.Movie
}

func (m Movie) IsValid() bool {
    /* do some checks and return true if it's valid... */
    return m.ID > 0
}
```

Iris is able to convert any custom data Structure into an HTTP Response Dispatcher,
so theoretically, something like the following is permitted if it's really necessary;

```go
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
    if !m.IsValid() {
        ctx.NotFound()
        return
    }
    ctx.JSON(m, context.JSON{Indent: " "})
}
```

However, we will use the "datamodels" as the only one models package because
Movie structure doesn't contain any sensitive data, clients are able to see all of its fields
and we don't need any extra functionality or validation inside it.

#### Controllers

Handles web requests, bridge between the services and the client.

```go
// file: web/controllers/movie_controller.go

package controllers

import (
    "errors"

    "github.com/kataras/iris/_examples/mvc/overview/datamodels"
    "github.com/kataras/iris/_examples/mvc/overview/services"

    "github.com/kataras/iris"
    "github.com/kataras/iris/mvc"
)

// MovieController is our /movies controller.
type MovieController struct {
    mvc.C

    // Our MovieService, it's an interface which
    // is binded from the main application.
    Service services.MovieService
}

// Get returns list of the movies.
// Demo:
// curl -i http://localhost:8080/movies
//
// The correct way if you have sensitive data:
// func (c *MovieController) Get() (results []viewmodels.Movie) {
//  data := c.Service.GetAll()
//
//  for _, movie := range data {
// 	  results = append(results, viewmodels.Movie{movie})
//  }
//  return
// }
// otherwise just return the datamodels.
func (c *MovieController) Get() (results []datamodels.Movie) {
    return c.Service.GetAll()
}

// GetBy returns a movie.
// Demo:
// curl -i http://localhost:8080/movies/1
func (c *MovieController) GetBy(id int64) (movie datamodels.Movie, found bool) {
    return c.Service.GetByID(id) // it will throw 404 if not found.
}

// PutBy updates a movie.
// Demo:
// curl -i -X PUT -F "genre=Thriller" -F "poster=@/Users/kataras/Downloads/out.gif" http://localhost:8080/movies/1
func (c *MovieController) PutBy(id int64) (datamodels.Movie, error) {
    // get the request data for poster and genre
    file, info, err := c.Ctx.FormFile("poster")
    if err != nil {
        return datamodels.Movie{}, errors.New("failed due form file 'poster' missing")
    }
    // we don't need the file so close it now.
    file.Close()

    // imagine that is the url of the uploaded file...
    poster := info.Filename
    genre := c.Ctx.FormValue("genre")

    return c.Service.UpdatePosterAndGenreByID(id, poster, genre)
}

// DeleteBy deletes a movie.
// Demo:
// curl -i -X DELETE -u admin:password http://localhost:8080/movies/1
func (c *MovieController) DeleteBy(id int64) interface{} {
    wasDel := c.Service.DeleteByID(id)
    if wasDel {
        // return the deleted movie's ID
        return iris.Map{"deleted": id}
    }
    // right here we can see that a method function can return any of those two types(map or int),
    // we don't have to specify the return type to a specific type.
    return iris.StatusBadRequest
}
```

```go
// file: web/controllers/hello_controller.go

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

```go
// file: web/middleware/basicauth.go

package middleware

import "github.com/kataras/iris/middleware/basicauth"

// BasicAuth middleware sample.
var BasicAuth = basicauth.New(basicauth.Config{
    Users: map[string]string{
        "admin": "password",
    },
})
```

```html
<!-- file: web/views/hello/index.html -->
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
<!-- file: web/views/hello/name.html -->
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

#### Main

This file creates any necessary component and links them together.

```go
// file: main.go

package main

import (
    "github.com/kataras/iris/_examples/mvc/overview/datasource"
    "github.com/kataras/iris/_examples/mvc/overview/repositories"
    "github.com/kataras/iris/_examples/mvc/overview/services"
    "github.com/kataras/iris/_examples/mvc/overview/web/controllers"
    "github.com/kataras/iris/_examples/mvc/overview/web/middleware"

    "github.com/kataras/iris"
)

func main() {
    app := iris.New()

    // Load the template files.
    app.RegisterView(iris.HTML("./web/views", ".html"))

    // Register our controllers.
    app.Controller("/hello", new(controllers.HelloController))

    // Create our movie repository with some (memory) data from the datasource.
    repo := repositories.NewMovieRepository(datasource.Movies)
    // Create our movie service, we will bind it to the movie controller.
    movieService := services.NewMovieService(repo)

    app.Controller("/movies", new(controllers.MovieController),
        // Bind the "movieService" to the MovieController's Service (interface) field.
        movieService,
        // Add the basic authentication(admin:password) middleware
        // for the /movies based requests.
        middleware.BasicAuth)

    // Start the web server at localhost:8080
    // http://localhost:8080/hello
    // http://localhost:8080/hello/iris
    // http://localhost:8080/movies
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

## Now you are ready to move to the next step and get closer to becoming a pro gopher

Congratulations, since you've made it so far, we've crafted just for you some next level content to turn you into a real pro gopher 😃

> Don't forget to prepare yourself a cup of coffee, or tea, whatever enjoys you the most!

* [Top 6 web frameworks for Go as of 2017](https://blog.usejournal.com/top-6-web-frameworks-for-go-as-of-2017-23270e059c4b)
* [Iris Go Framework + MongoDB](https://medium.com/go-language/iris-go-framework-mongodb-552e349eab9c)
* [How to build a file upload form using DropzoneJS and Go](https://hackernoon.com/how-to-build-a-file-upload-form-using-dropzonejs-and-go-8fb9f258a991)
* [How to display existing files on server using DropzoneJS and Go](https://hackernoon.com/how-to-display-existing-files-on-server-using-dropzonejs-and-go-53e24b57ba19)
* [Iris, a modular web framework](https://medium.com/@corebreaker/iris-web-cd684b4685c7)
* [Go vs .NET Core in terms of HTTP performance](https://medium.com/@kataras/go-vs-net-core-in-terms-of-http-performance-7535a61b67b8)
* [Iris Go vs .NET Core Kestrel in terms of HTTP performance](https://hackernoon.com/iris-go-vs-net-core-kestrel-in-terms-of-http-performance-806195dc93d5)
* [How to Turn an Android Device into a Web Server](https://twitter.com/ThePracticalDev/status/892022594031017988)
* [Deploying a Iris Golang app in hasura](https://medium.com/@HasuraHQ/deploy-an-iris-golang-app-with-backend-apis-in-minutes-25a559bf530b)
* [A URL Shortener Service using Go, Iris and Bolt](https://medium.com/@kataras/a-url-shortener-service-using-go-iris-and-bolt-4182f0b00ae7)

## People

The author of Iris is [@kataras](https://github.com/kataras), you can reach him via;

* [Medium](https://medium.com/@kataras)
* [Twitter](https://twitter.com/makismaropoulos)
* [Dev.to](https://dev.to/@kataras)
* [Facebook](https://facebook.com/iris.framework)
* [Mail](mailto:kataras2006@hotmail.com?subject=Iris%20I%20need%20some%20help%20please)

[List of all Authors](AUTHORS)

[List of all Contributors](https://github.com/kataras/iris/graphs/contributors)

Help this project to continue deliver awesome and unique features with the higher code quality as possible by donating any amount via [PayPal](https://www.paypal.me/kataras) or [BTC](https://iris-go.com/v8/donate).

For more information about contributing to the Iris project please check the [CONTRIBUTING.md file](CONTRIBUTING.md).

### We need your help with translations into your native language

Iris needs your help, please think about contributing to the translation of the [README](README.md) and https://iris-go.com, you will be rewarded.

Instructions can be found at: https://github.com/kataras/iris/issues/796

### 03, October 2017 | Iris User Experience Report

Be part of the **first** Iris User Experience Report by submitting a simple form, it won't take more than **2 minutes**.

The form contains some questions that you may need to answer in order to learn more about you; learning more about you helps us to serve you with the best possible way!

https://docs.google.com/forms/d/e/1FAIpQLSdCxZXPANg_xHWil4kVAdhmh7EBBHQZ_4_xSZVDL-oCC_z5pA/viewform?usp=sf_link

## Sponsors

Support this project by becoming a sponsor. Your logo will show up here with a link to your website. [Become a sponsor](https://opencollective.com/iris#sponsor)

<a href="https://opencollective.com/iris/sponsor/0/website" target="_blank"><img src="https://opencollective.com/iris/sponsor/0/avatar.svg"></a>
<a href="https://opencollective.com/iris/sponsor/1/website" target="_blank"><img src="https://opencollective.com/iris/sponsor/1/avatar.svg"></a>
<a href="https://opencollective.com/iris/sponsor/2/website" target="_blank"><img src="https://opencollective.com/iris/sponsor/2/avatar.svg"></a>
<a href="https://opencollective.com/iris/sponsor/3/website" target="_blank"><img src="https://opencollective.com/iris/sponsor/3/avatar.svg"></a>
<a href="https://opencollective.com/iris/sponsor/4/website" target="_blank"><img src="https://opencollective.com/iris/sponsor/4/avatar.svg"></a>
<a href="https://opencollective.com/iris/sponsor/5/website" target="_blank"><img src="https://opencollective.com/iris/sponsor/5/avatar.svg"></a>
<a href="https://opencollective.com/iris/sponsor/6/website" target="_blank"><img src="https://opencollective.com/iris/sponsor/6/avatar.svg"></a>
<a href="https://opencollective.com/iris/sponsor/7/website" target="_blank"><img src="https://opencollective.com/iris/sponsor/7/avatar.svg"></a>
<a href="https://opencollective.com/iris/sponsor/8/website" target="_blank"><img src="https://opencollective.com/iris/sponsor/8/avatar.svg"></a>
<a href="https://opencollective.com/iris/sponsor/9/website" target="_blank"><img src="https://opencollective.com/iris/sponsor/9/avatar.svg"></a>

## License

Iris is licensed under the 3-Clause BSD [License](LICENSE). Iris is 100% open-source software.

For any questions regarding the license please [contact us](mailto:kataras2006@hotmail.com?subject=Iris%20License).
