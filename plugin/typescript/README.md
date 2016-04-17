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
1.  Bin:    string, the typescript installation path/bin/tsc or tsc.cmd, if empty then it will search to the global npm modules
2. Dir:     string, Dir set the root, where to search for typescript files/project. Default "./"
3. Ignore:  string, comma separated ignore typescript files/project from these directories. Default "" (node_modules are always ignored)
4. Watch:	 boolean, watch for any changes and re-build if true/. Default true -- This is not ready yet

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
	Watch	->        watch for any changes and re-build if true/. Default true -- This is not ready yet
	*/

	ts := typescript.Options {
		Dir: "./scripts/src"
	}

	//if you want to change only certain option(s) but you want default to all others then you have to do this:
	ts = typescript.DefaultOptions()
	ts.Watch = false
	//

	iris.Plugin(typescript.New(ts)) //or with the default options just: typescript.New()

	iris.Get("/", func (ctx *iris.Context){})

	iris.Listen()
}


```

