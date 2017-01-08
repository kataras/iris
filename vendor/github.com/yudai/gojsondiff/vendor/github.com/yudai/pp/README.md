# pp [![wercker status](https://app.wercker.com/status/6934c847631da2cf672e559f927a90b2/s "wercker status")](https://app.wercker.com/project/bykey/6934c847631da2cf672e559f927a90b2)

Colored pretty printer for Go language

![](http://i.gyazo.com/d3253ae839913b7239a7229caa4af551.png)

## Usage

Just call `pp.Print()`.

```go
import "github.com/k0kubun/pp"

m := map[string]string{"foo": "bar", "hello": "world"}
pp.Print(m)
```

![](http://i.gyazo.com/0d08376ed2656257627f79626d5e0cde.png)

### API

fmt package-like functions are provided.

```go
pp.Print()
pp.Println()
pp.Sprint()
pp.Fprintf()
// ...
```

API doc is available at: http://godoc.org/github.com/k0kubun/pp

## Demo

### Timeline

![](http://i.gyazo.com/a8adaeec965db943486e35083cf707f2.png)

### UserStream event

![](http://i.gyazo.com/1e88915b3a6a9129f69fb5d961c4f079.png)

### Works on windows

![](http://i.gyazo.com/ab791997a980f1ab3ee2a01586efdce6.png)

## License

MIT License
