# Plugins

Navigate through [iris-contrib/plugin](https://github.com/iris-contrib/plugin) repository to view all available 'plugins'.

>  By the word 'plugin', we mean an event-driven system and not the future go1.8 plugin feature.


## Installation

```sh
$ go get github.com/iris-contrib/plugin/...
```

## How can I register a plugin?

```go
app := iris.New()
app.Plugins.Add(thePlugin)
```
