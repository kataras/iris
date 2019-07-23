# Jet Engine Example (Embedded)

Take a look at the [../template_jet_0](../template_jet_0)'s README first.

This example teaches you how to use jet templates embedded in your applications with ease using the Iris built-in Jet view engine.

This example is a customized fork of https://github.com/CloudyKit/jet/tree/master/examples/asset_packaging, so you can
notice the differences side by side. For example, you don't have to use any external package inside your application,
Iris manually builds the template loader for binary data when Asset and AssetNames are available through tools like the [go-bindata](github.com/shuLhan/go-bindata).

Note that you can still use any custom loaders through the `JetEngine.SetLoader`
which overrides any previous loaders like `JetEngine.Binary` we use on this example.

## How to run

```sh
$ go get -u github.com/shuLhan/go-bindata/... # or any active alternative
$ go-bindata ./views/...
$ go build
$ ./template_jet_0_embedded
```

Repeat the above steps on any `./views` changes.

> html files are not used, only binary data. You can move or delete the `./views` folder.
