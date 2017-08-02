First of all, install [quicktemplate](https://github.com/valyala/quicktemplate) package and [quicktemplate compiler](https://github.com/valyala/quicktemplate/tree/master/qtc)

```sh
go get -u github.com/valyala/quicktemplate
go get -u github.com/valyala/quicktemplate/qtc
```

The example has the Go code compiled already for you, therefore:
```sh
go run main.go # http://localhost:8080
```

However there is an instruction below, full documentation can be found at https://github.com/valyala/quicktemplate.

Save your template files into `templates` folder under the extension *.qtpl, open your terminal and run `qtc` inside this folder.

If all went ok, `*.qtpl.go` files must appear in the `templates` folder. These files contain the Go code for  all `*.qtpl` files.

> Remember, each time you change a  a `/templates/*.qtpl` file you have to run the `qtc` command and re-build your application.