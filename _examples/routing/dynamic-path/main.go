package main

import (
	"regexp"

	"github.com/kataras/iris/v12"
)

func main() {
	app := iris.New()

	// At the previous example "routing/basic",
	// we've seen static routes, group of routes, subdomains, wildcard subdomains, a small example of parameterized path
	// with a single known paramete and custom http errors, now it's time to see wildcard parameters and macros.

	// Iris, like net/http std package registers route's handlers
	// by a Handler, the iris' type of handler is just a func(ctx iris.Context)
	// where context comes from github.com/kataras/iris/context.
	//
	// Iris has the easiest and the most powerful routing process you have ever meet.
	//
	// At the same time,
	// Iris has its own interpeter(yes like a programming language)
	// for route's path syntax and their dynamic path parameters parsing and evaluation,
	// We call them "macros" for shortcut.
	// How? It calculates its needs and if not any special regexp needed then it just
	// registers the route with the low-level underline  path syntax,
	// otherwise it pre-compiles the regexp and adds the necessary middleware(s).
	//
	// Standard macro types for parameters:
	// +------------------------+
	// | {param:string}         |
	// +------------------------+
	// string type
	// anything (single path segmnent)
	//
	// +-------------------------------+
	// | {param:int}                   |
	// +-------------------------------+
	// int type
	// -9223372036854775808 to 9223372036854775807 (x64) or -2147483648 to 2147483647 (x32), depends on the host arch
	//
	// +------------------------+
	// | {param:int8}           |
	// +------------------------+
	// int8 type
	// -128 to 127
	//
	// +------------------------+
	// | {param:int16}          |
	// +------------------------+
	// int16 type
	// -32768 to 32767
	//
	// +------------------------+
	// | {param:int32}          |
	// +------------------------+
	// int32 type
	// -2147483648 to 2147483647
	//
	// +------------------------+
	// | {param:int64}          |
	// +------------------------+
	// int64 type
	// -9223372036854775808 to 9223372036854775807
	//
	// +------------------------+
	// | {param:uint}           |
	// +------------------------+
	// uint type
	// 0 to 18446744073709551615 (x64) or 0 to 4294967295 (x32)
	//
	// +------------------------+
	// | {param:uint8}          |
	// +------------------------+
	// uint8 type
	// 0 to 255
	//
	// +------------------------+
	// | {param:uint16}         |
	// +------------------------+
	// uint16 type
	// 0 to 65535
	//
	// +------------------------+
	// | {param:uint32}          |
	// +------------------------+
	// uint32 type
	// 0 to 4294967295
	//
	// +------------------------+
	// | {param:uint64}         |
	// +------------------------+
	// uint64 type
	// 0 to 18446744073709551615
	//
	// +---------------------------------+
	// | {param:bool} or {param:boolean} |
	// +---------------------------------+
	// bool type
	// only "1" or "t" or "T" or "TRUE" or "true" or "True"
	// or "0" or "f" or "F" or "FALSE" or "false" or "False"
	//
	// +------------------------+
	// | {param:alphabetical}   |
	// +------------------------+
	// alphabetical/letter type
	// letters only (upper or lowercase)
	//
	// +------------------------+
	// | {param:file}           |
	// +------------------------+
	// file type
	// letters (upper or lowercase)
	// numbers (0-9)
	// underscore (_)
	// dash (-)
	// point (.)
	// no spaces ! or other character
	//
	// +------------------------+
	// | {param:path}           |
	// +------------------------+
	// path type
	// anything, should be the last part, can be more than one path segment,
	// i.e: "/test/{param:path}" and request: "/test/path1/path2/path3" , ctx.Params().Get("param") == "path1/path2/path3"
	//
	// +------------------------+
	// | {param:uuid}           |
	// +------------------------+
	// UUIDv4 (and v1) path parameter validation.
	//
	// +------------------------+
	// | {param:mail}           |
	// +------------------------+
	// Email without domain validation.
	//
	// +------------------------+
	// | {param:email}           |
	// +------------------------+
	// Email with domain validation.
	//
	//
	// +------------------------+
	// | {param:date}           |
	// +------------------------+
	// yyyy/mm/dd format e.g. /blog/{param:date} matches /blog/2022/04/21.
	//
	// +------------------------+
	// | {param:weekday}        |
	// +------------------------+
	// positive integer 0 to 6 or
	// string of time.Weekday longname format ("sunday" to "monday" or "Sunday" to "Monday")
	// format e.g. /schedule/{param:weekday} matches /schedule/monday.
	//
	// If type is missing then parameter's type is defaulted to string, so
	// {param} is identical to {param:string}.
	//
	// If a function not found on that type then the `string` macro type's functions are being used.
	//
	//
	// Besides the fact that iris provides the basic types and some default "macro funcs"
	// you are able to register your own too!.
	//
	// Register a named path parameter function:
	// app.Macros().Number.RegisterFunc("min", func(argument int) func(paramValue string) bool {
	//  [...]
	//  return true/false -> true means valid.
	// })
	//
	// at the func(argument ...) you can have any standard type, it will be validated before the server starts
	// so don't care about performance here, the only thing it runs at serve time is the returning func(paramValue string) bool.
	//
	// {param:string equal(iris)} , "iris" will be the argument here:
	// app.Macros().String.RegisterFunc("equal", func(argument string) func(paramValue string) bool {
	// 	return func(paramValue string) bool { return argument == paramValue }
	// })

	// Optionally, set custom handler on path parameter type error:
	app.Macros().Get("uuid").HandleError(func(ctx iris.Context, paramIndex int, err error) {
		ctx.StatusCode(iris.StatusBadRequest)

		param := ctx.Params().GetEntryAt(paramIndex)
		ctx.JSON(iris.Map{
			"error":     err.Error(),
			"message":   "invalid path parameter",
			"parameter": param.Key,
			"value":     param.ValueRaw,
		})
	})

	// http://localhost:8080/user/bb4f33e4-dc08-40d8-9f2b-e8b2bb615c0e -> OK
	// http://localhost:8080/user/dsadsa-invalid-uuid                  -> NOT FOUND
	app.Get("/user/{id:uuid}", func(ctx iris.Context) {
		id := ctx.Params().Get("id")
		ctx.WriteString(id)
	})

	// +------------------------+
	// | {param:email}           |
	// +------------------------+
	// Email + mx look uppath parameter validation.
	// Note that, you can also use the simpler ":mail" to accept any domain email.

	// http://localhost:8080/user/email/kataras2006@hotmail.com -> OK
	// http://localhost:8080/user/email/b-c@                    -> NOT FOUND
	app.Get("/user/email/{user_email:email}", func(ctx iris.Context) {
		email := ctx.Params().Get("user_email")
		ctx.WriteString(email)
	})

	// http://localhost:8080/blog/2022/04/21
	app.Get("/blog/{date:date}", func(ctx iris.Context) {
		// rawTimeValue := ctx.Params().GetEntry("d").ValueRaw.(time.Time)
		// OR
		rawTimeValue, _ := ctx.Params().GetTime("date")
		// yearMonthDay := rawTimeValue.Format("2006/01/02")
		// OR
		yearMonthDay := ctx.Params().SimpleDate("date")
		ctx.Writef("Raw time.Time.String value: %v\nyyyy/mm/dd: %s\n", rawTimeValue, yearMonthDay)
	})

	// 0 to 7 or "Sunday" to "Monday" or "sunday" to "monday". Leading zeros don't matter.
	// http://localhost:8080/schedule/monday or http://localhost:8080/schedule/Monday or
	// http://localhost:8080/schedule/1 or http://localhost:8080/schedule/0001.
	app.Get("/schedule/{day:weekday}", func(ctx iris.Context) {
		day, _ := ctx.Params().GetWeekday("day")
		ctx.Writef("Weekday requested was: %v\n", day)
	})

	// you can use the "string" type which is valid for a single path parameter that can be anything.
	app.Get("/username/{name}", func(ctx iris.Context) {
		ctx.Writef("Hello %s", ctx.Params().Get("name"))
	}) // type is missing = {name:string}

	// Let's register our first macro attached to uint64 macro type.
	// "min" = the function
	// "minValue" = the argument of the function
	// func(uint64) bool = our func's evaluator, this executes in serve time when
	// a user requests a path which contains the :uint64 macro parameter type with the min(...) macro parameter function.
	app.Macros().Get("uint64").RegisterFunc("min", func(minValue uint64) func(uint64) bool {
		// type of "paramValue" should match the type of the internal macro's evaluator function, which in this case is "uint64".
		return func(paramValue uint64) bool {
			return paramValue >= minValue
		}
	})

	// http://localhost:8080/profile/id>=20
	// this will throw 404 even if it's found as route on : /profile/0, /profile/blabla, /profile/-1
	// macro parameter functions are optional of course.
	app.Get("/profile/{id:uint64 min(20)}", func(ctx iris.Context) {
		// second parameter is the error but it will always nil because we use macros,
		// the validaton already happened.
		id := ctx.Params().GetUint64Default("id", 0)
		ctx.Writef("Hello id: %d", id)
	})

	// to change the error code per route's macro evaluator:
	app.Get("/profile/{id:uint64 min(1)}/friends/{friendid:uint64 min(1) else 504}", func(ctx iris.Context) {
		id := ctx.Params().GetUint64Default("id", 0)
		friendid := ctx.Params().GetUint64Default("friendid", 0)
		ctx.Writef("Hello id: %d looking for friend id: %d", id, friendid)
	}) // this will throw e 504 error code instead of 404 if all route's macros not passed.

	// :uint8 0 to 255.
	app.Get("/ages/{age:uint8 else 400}", func(ctx iris.Context) {
		age, _ := ctx.Params().GetUint8("age")
		ctx.Writef("age selected: %d", age)
	})

	// Another example using a custom regexp or any custom logic.

	// Register your custom argument-less macro function to the :string param type.
	latLonExpr := "^-?[0-9]{1,3}(?:\\.[0-9]{1,10})?$"
	latLonRegex, err := regexp.Compile(latLonExpr)
	if err != nil {
		panic(err)
	}

	// MatchString is a type of func(string) bool, so we use it as it is.
	app.Macros().Get("string").RegisterFunc("coordinate", latLonRegex.MatchString)

	app.Get("/coordinates/{lat:string coordinate() else 502}/{lon:string coordinate() else 502}", func(ctx iris.Context) {
		ctx.Writef("Lat: %s | Lon: %s", ctx.Params().Get("lat"), ctx.Params().Get("lon"))
	})

	//

	// Another one is by using a custom body.
	app.Macros().Get("string").RegisterFunc("range", func(minLength, maxLength int) func(string) bool {
		return func(paramValue string) bool {
			return len(paramValue) >= minLength && len(paramValue) <= maxLength
		}
	})

	app.Get("/limitchar/{name:string range(1,200)}", func(ctx iris.Context) {
		name := ctx.Params().Get("name")
		ctx.Writef(`Hello %s | the name should be between 1 and 200 characters length
		otherwise this handler will not be executed`, name)
	})

	//

	// Register your custom macro function which accepts a slice of strings `[...,...]`.
	app.Macros().Get("string").RegisterFunc("has", func(validNames []string) func(string) bool {
		return func(paramValue string) bool {
			for _, validName := range validNames {
				if validName == paramValue {
					return true
				}
			}

			return false
		}
	})

	app.Get("/static_validation/{name:string has([kataras,gerasimos,maropoulos])}", func(ctx iris.Context) {
		name := ctx.Params().Get("name")
		ctx.Writef(`Hello %s | the name should be "kataras" or "gerasimos" or "maropoulos"
		otherwise this handler will not be executed`, name)
	})

	//

	// http://localhost:8080/game/a-zA-Z/level/42
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

	// http://localhost:8080/trimmed/42.html
	app.Get("/trimmed/{uid:string regexp(^[0-9]{1,20}.html$)}", iris.TrimParamFilePart, func(ctx iris.Context) {
		//
		// The above line is useless now that we've registered the TrimParamFilePart middleware:
		// uid := ctx.Params().GetTrimFileUint64("uid")
		// TrimParamFilePart can be registered to a Party (group of routes) too.

		uid := ctx.Params().GetUint64Default("uid", 0)
		ctx.Writef("Param value: %d\n", uid)
	})

	// "{param}"'s performance is exactly the same of ":param"'s.

	// alternatives -> ":param" for single path parameter and "*" for wildcard path parameter.
	// Note these:
	// if  "/mypath/*" then the parameter name is "*".
	// if  "/mypath/{myparam:path}" then the parameter has two names, one is the "*" and the other is the user-defined "myparam".

	// WARNING:
	// A path parameter name should contain only alphabetical letters or digits. Symbols like  '_' are NOT allowed.
	// Last, do not confuse `ctx.Params()` with `ctx.Values()`.
	// Path parameter's values can be retrieved from `ctx.Params()`,
	// context's local storage that can be used to communicate between handlers and middleware(s) can be stored to `ctx.Values()`.
	//
	// When registering different parameter types in the same exact path pattern, the path parameter's name
	// should differ e.g.
	// /path/{name:string}
	// /path/{id:uint}
	//
	// Note:
	// If * path part is declared at the end of the route path, then
	// it's considered a wildcard (same as {p:path}). In order to declare
	// literal * and over pass this limitation use the string's path parameter 'eq' function
	// as shown below:
	// app.Get("/*/*/{p:string eq(*)}", handler) <- This will match only: /*/*/* and not /*/*/anything.
	app.Listen(":8080")
}
