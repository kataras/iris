## Package information

This package is the command line tool for  [../](https://github.com/kataras/iris).

[![Iris help screen](https://raw.githubusercontent.com/iris-contrib/website/gh-pages/assets/iris_cli_screen.png)](https://raw.githubusercontent.com/iris-contrib/website/gh-pages/assets/iris_cli_screen.png)

[![Iris installed screen](https://raw.githubusercontent.com/iris-contrib/website/gh-pages/assets/iris_cli_screen2.png)](https://raw.githubusercontent.com/iris-contrib/website/gh-pages/assets/iris_cli_screen2.png)

## Install
Current version: 0.0.7
```sh

go get -u github.com/kataras/iris/iris

```

# Usage


```sh
$ iris [command] [-flags]
```

> Note that you must have $GOPATH/bin to your $PATH system/environment variable.


## create


**The create command** creates for you a start project in a directory


```sh
iris create -t basic -d myprojects/iris1
```

Will create the  [basic](https://github.com/iris-contrib/iris-command-assets/tree/master/basic) sample package to the `$GOPATH/src/myprojects/iris1` directory and run the app.

```sh
iris create -t static  -d myprojects/iris1
```

Will create the [static](https://github.com/iris-contrib/iris-command-assets/tree/master/static) sample package to the `$GOPATH/src/myprojects/iris1` directory and run the app.


The default

```sh
iris create
```

Will create the basic sample package to `$GOPATH/src/myiris` directory and run the app.

```sh
iris create -d myproject
```

Will create the basic sample package to the `$GOPATH/src/myproject` folder and run the app.

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
