## Package information

This package is the command line tool for  [../](https://github.com/kataras/iris).


[![Iris get command preview](https://raw.githubusercontent.com/iris-contrib/website/gh-pages/assets/iriscmd.gif)](https://raw.githubusercontent.com/iris-contrib/website/gh-pages/assets/iriscmd.gif)


[![Iris help screen](https://raw.githubusercontent.com/iris-contrib/website/gh-pages/assets/iris_cli_screen.png)](https://raw.githubusercontent.com/iris-contrib/website/gh-pages/assets/iris_cli_screen.png)


## Install
```sh

go get -u github.com/kataras/iris/iris

```

# Usage


```sh
$ iris [command] [-flags]
```

> Note that you must have $GOPATH/bin to your $PATH system/environment variable.


## get


**The get command** downloads, installs and runs a project based on a `prototype`, such as `basic`, `static` and `mongo` .

> These projects are located [online](https://github.com/iris-contrib/examples/tree/master/AIO_examples)


```sh
iris get mongo
```

Downloads the  [mongo](https://github.com/iris-contrib/examples/tree/master/AIO_examples/mongo) sample protoype project to the `$GOPATH/src/github.com/iris-contrib/examples` directory(the iris cmd will open this folder to you, automatically) builds, runs and watch for source code changes (hot-reload)


## run

**The run command** runs & reload on file changes your Iris station

It's like ` go run ` but with directory watcher and re-run on .go file changes.

```sh
iris run main.go
```

[![Iris CLI run showcase](https://raw.githubusercontent.com/iris-contrib/website/gh-pages/assets/iris_command_line_tool_run_command.png)](https://raw.githubusercontent.com/iris-contrib/website/gh-pages/assets/iris_command_line_tool_run_command.png)

[![Iris CLI run showcase linux](https://raw.githubusercontent.com/iris-contrib/website/gh-pages/assets/iris_command_line_tool_run_linux.png)](https://raw.githubusercontent.com/iris-contrib/website/gh-pages/assets/iris_command_line_tool_run_linux.png)


## version

```sh
iris version
```

Will print the current Iris' installed version to your machine
