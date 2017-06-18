package main

import (
	"github.com/cdren/iris"
	"github.com/cdren/iris/typescript"
)

// NOTE: Some machines don't allow to install typescript automatically, so if you don't have typescript installed
// and the typescript adaptor doesn't works for you then follow the below steps:
// 1. close the iris server
// 2. open your terminal and execute: npm install -g typescript
// 3. start your iris server, it should be work, as expected, now.
func main() {
	app := iris.New()

	ts := typescript.New()
	ts.Config.Dir = "./www/scripts"
	ts.Attach(app) // attach the application to the typescript compiler

	app.StaticWeb("/", "./www") // serve the index.html
	app.Run(iris.Addr(":8080"))
}

// open http://localhost:8080
// go to ./www/scripts/app.ts
// make a change
// reload the http://localhost:8080 and you should see the changes
//
// what it does?
// - compiles the typescript files using default compiler options if not tsconfig found
// - watches for changes on typescript files, if a change then it recompiles the .ts to .js
//
// same as you used to do with gulp-like tools, but here at Iris I do my bests to help GO developers.
