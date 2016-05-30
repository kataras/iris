## Package information

This is an Iris and typescript bridge plugin.

1. Search for typescript files (.ts)
2. Search for typescript projects (.tsconfig)
3. If 1 || 2 continue else stop
4. Check if typescript is installed, if not then auto-install it (always inside npm global modules, -g)
5. If typescript project then build the project using tsc -p $dir
6. If typescript files and no project then build each typescript using tsc $filename
7. Watch typescript files if any changes happens, then re-build (5|6)

> Note: Ignore all typescript files & projects whose path has '/node_modules/'


## Options

This plugin has **optionally** options
1. Bin:    string, the typescript installation path/bin/tsc or tsc.cmd, if empty then it will search to the global npm modules
2. Dir:     string, Dir set the root, where to search for typescript files/project. Default "./"
3. Ignore:  string, comma separated ignore typescript files/project from these directories. Default "" (node_modules are always ignored)
4. Tsconfig:  &typescript.Tsconfig{}, here you can set all compilerOptions if no tsconfig.json exists inside the 'Dir'
5. Editor: 	typescript.Editor(), if setted then alm-tools browser-based typescript IDE will be available. Defailt is nil

> Note: if any string in Ignore doesn't start with './' then it will ignore all files which contains this path string.
For example /node_modules/ will ignore all typescript files that are inside at ANY '/node_modules/', that means and the submodules.


## How to use

```go

package main

import (
	"github.com/kataras/iris"
	"github.com/kataras/iris/plugin/typescript"
)

func main(){
	/* Options
	Bin ->  		  the typescript installation path/bin/tsc or tsc.cmd, if empty then it will search to the global npm modules
	Dir    ->		   where to search for typescript files/project. Default "./"
	Ignore ->        comma separated ignore typescript files/project from these directories (/node_modules/ are always ignored). Default ""
    Tsconfig -> 		&typescript.Tsconfig{}, here you can set all compilerOptions if no tsconfig.json exists inside the 'Dir'
    Editor ->			typescript.Editor(), if setted then alm-tools browser-based typescript IDE will be available. Default is nil.
	*/

	ts := typescript.Options {
		Dir: "./scripts/src",
		Tsconfig: &typescript.Tsconfig{Module: "commonjs", Target: "es5"}, // or typescript.DefaultTsconfig()
	}

	//if you want to change only certain option(s) but you want default to all others then you have to do this:
	ts = typescript.DefaultOptions()
	//

	iris.Plugins().Add(typescript.New(ts)) //or with the default options just: typescript.New()

	iris.Get("/", func (ctx *iris.Context){})

	iris.Listen(":8080")
}


```

## Editor

[alm-tools](http://alm.tools) is a typescript online IDE/Editor, made by [@basarat](https://twitter.com/basarat) one of the top contributors of the [Typescript](http://www.typescriptlang.org).

Iris gives you the opportunity to edit your client-side using the alm-tools editor, via the editor plugin.
With typescript plugin you have to set the Editor option and you're ready:

```go
typescript.Options {
	//...
	Editor: typescript.Editor("username","passowrd")
	//...
}
```

> [Read more](https://github.com/kataras/iris/tree/development/plugin/editor) for Editor
