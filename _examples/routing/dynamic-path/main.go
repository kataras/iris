package main

import (
	"strconv"

	"github.com/kataras/iris"
)

func main() {
	app := iris.New()

	// At the previous example "routing/basic",
	// we've seen static routes, group of routes, subdomains, wildcard subdomains, a small example of parameterized path
	// with a single known paramete and custom http errors, now it's time to see wildcard parameters and macros.

	// iris, like net/http std package registers route's handlers
	// by a Handler, the iris' type of handler is just a func(ctx iris.Context)
	// where context comes from github.com/kataras/iris/context.
	// Until go 1.9 you will have to import that package too, after go 1.9 this will be not be necessary.
	//
	// iris has the easiest and the most powerful routing process you have ever meet.
	//
	// At the same time,
	// iris has its own interpeter(yes like a programming language)
	// for route's path syntax and their dynamic path parameters parsing and evaluation,
	// We call them "macros" for shortcut.
	// How? It calculates its needs and if not any special regexp needed then it just
	// registers the route with the low-level underline  path syntax,
	// otherwise it pre-compiles the regexp and adds the necessary middleware(s).
	//
	// Standard macro types for parameters:
	//  +------------------------+
	//  | {param:string}         |
	//  +------------------------+
	// string type
	// anything
	//
	//  +------------------------+
	//  | {param:int}            |
	//  +------------------------+
	// int type
	// only numbers (0-9)
	//
	// +------------------------+
	// | {param:long}           |
	// +------------------------+
	// int64 type
	// only numbers (0-9)
	//
	// +------------------------+
	// | {param:boolean}        |
	// +------------------------+
	// bool type
	// only "1" or "t" or "T" or "TRUE" or "true" or "True"
	// or "0" or "f" or "F" or "FALSE" or "false" or "False"
	//
	//  +------------------------+
	//  | {param:alphabetical}   |
	//  +------------------------+
	// alphabetical/letter type
	// letters only (upper or lowercase)
	//
	//  +------------------------+
	//  | {param:file}           |
	//  +------------------------+
	// file type
	// letters (upper or lowercase)
	// numbers (0-9)
	// underscore (_)
	// dash (-)
	// point (.)
	// no spaces ! or other character
	//
	//  +------------------------+
	//  | {param:path}           |
	//  +------------------------+
	// path type
	// anything, should be the last part, more than one path segment,
	// i.e: /path1/path2/path3 , ctx.Params().Get("param") == "/path1/path2/path3"
	//
	// if type is missing then parameter's type is defaulted to string, so
	// {param} == {param:string}.
	//
	// If a function not found on that type then the `string` macro type's functions are being used.
	//
	//
	// Besides the fact that iris provides the basic types and some default "macro funcs"
	// you are able to register your own too!.
	//
	// Register a named path parameter function:
	// app.Macros().Int.RegisterFunc("min", func(argument int) func(paramValue string) bool {
	//  [...]
	//  return true/false -> true means valid.
	// })
	//
	// at the func(argument ...) you can have any standard type, it will be validated before the server starts
	// so don't care about performance here, the only thing it runs at serve time is the returning func(paramValue string) bool.
	//
	// {param:string equal(iris)} , "iris" will be the argument here:
	// app.Macros().String.RegisterFunc("equal", func(argument string) func(paramValue string) bool {
	//  return func(paramValue string){ return argument == paramValue }
	// })

	// you can use the "string" type which is valid for a single path parameter that can be anything.
	app.Get("/username/{name}", func(ctx iris.Context) {
		ctx.Writef("Hello %s", ctx.Params().Get("name"))
	}) // type is missing = {name:string}

	// Let's register our first macro attached to int macro type.
	// "min" = the function
	// "minValue" = the argument of the function
	// func(string) bool = the macro's path parameter evaluator, this executes in serve time when
	// a user requests a path which contains the :int macro type with the min(...) macro parameter function.
	app.Macros().Int.RegisterFunc("min", func(minValue int) func(string) bool {
		// do anything before serve here [...]
		// at this case we don't need to do anything
		return func(paramValue string) bool {
			n, err := strconv.Atoi(paramValue)
			if err != nil {
				return false
			}
			return n >= minValue
		}
	})

	// http://localhost:8080/profile/id>=1
	// this will throw 404 even if it's found as route on : /profile/0, /profile/blabla, /profile/-1
	// macro parameter functions are optional of course.
	app.Get("/profile/{id:int min(1)}", func(ctx iris.Context) {
		// second parameter is the error but it will always nil because we use macros,
		// the validaton already happened.
		id, _ := ctx.Params().GetInt("id")
		ctx.Writef("Hello id: %d", id)
	})

	// to change the error code per route's macro evaluator:
	app.Get("/profile/{id:int min(1)}/friends/{friendid:int min(1) else 504}", func(ctx iris.Context) {
		id, _ := ctx.Params().GetInt("id")
		friendid, _ := ctx.Params().GetInt("friendid")
		ctx.Writef("Hello id: %d looking for friend id: ", id, friendid)
	}) // this will throw e 504 error code instead of 404 if all route's macros not passed.

	// http://localhost:8080/game/a-zA-Z/level/0-9
	// remember, alphabetical is lowercase or uppercase letters only.
	app.Get("/game/{name:alphabetical}/level/{level:int}", func(ctx iris.Context) {
		ctx.Writef("name: %s | level: %s", ctx.Params().Get("name"), ctx.Params().Get("level"))
	})

	app.Get("/lowercase/static", func(ctx iris.Context) {
		ctx.Writef("static and dynamic paths are not conflicted anymore!")
	})

	// let's use a trivial custom regexp that validates a single path parameter
	// which its value is only lowercase letters.

	// http://localhost:8080/lowercase/anylowercase
	app.Get("/lowercase/{name:string regexp(^[a-z]+)}", func(ctx iris.Context) {
		ctx.Writef("name should be only lowercase, otherwise this handler will never executed: %s", ctx.Params().Get("name"))
	})

	// http://localhost:8080/single_file/app.js
	app.Get("/single_file/{myfile:file}", func(ctx iris.Context) {
		ctx.Writef("file type validates if the parameter value has a form of a file name, got: %s", ctx.Params().Get("myfile"))
	})

	// http://localhost:8080/myfiles/any/directory/here/
	// this is the only macro type that accepts any number of path segments.
	app.Get("/myfiles/{directory:path}", func(ctx iris.Context) {
		ctx.Writef("path type accepts any number of path segments, path after /myfiles/ is: %s", ctx.Params().Get("directory"))
	}) // for wildcard path (any number of path segments) without validation you can use:
	// /myfiles/*

	// "{param}"'s performance is exactly the same of ":param"'s.

	// alternatives -> ":param" for single path parameter and "*" for wildcard path parameter.
	// Note these:
	// if  "/mypath/*" then the parameter name is "*".
	// if  "/mypath/{myparam:path}" then the parameter has two names, one is the "*" and the other is the user-defined "myparam".

	// WARNING:
	// A path parameter name should contain only alphabetical letters. Symbols like  '_' and numbers are NOT allowed.
	// Last, do not confuse `ctx.Params()` with `ctx.Values()`.
	// Path parameter's values goes to `ctx.Params()` and context's local storage
	// that can be used to communicate between handlers and middleware(s) goes to
	// `ctx.Values()`.
	app.Run(iris.Addr(":8080"))
}
