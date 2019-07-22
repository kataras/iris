# Iris Web Framework <a href="README_ZH.md"> <img width="20px" src="https://iris-go.com/images/flag-china.svg?v=10" /></a> <a href="README_RU.md"><img width="20px" src="https://iris-go.com/images/flag-russia.svg?v=10" /></a> <a href="README_ID.md"> <img width="20px" src="https://iris-go.com/images/flag-indonesia.svg?v=10" /></a> <a href="README_GR.md"><img width="20px" src="https://iris-go.com/images/flag-greece.svg?v=10" /></a> <a href="README_PT_BR.md"><img width="20px" src="https://iris-go.com/images/flag-pt-br.svg?v=10" /></a> <a href="README_JPN.md"><img width="20px" src="https://iris-go.com/images/flag-japan.svg?v=10" /></a>

[![build status](https://img.shields.io/travis/kataras/iris/master.svg?style=for-the-badge)](https://travis-ci.org/kataras/iris) [![report card](https://img.shields.io/badge/report%20card-a%2B-ff3333.svg?style=for-the-badge)](https://goreportcard.com/report/github.com/kataras/iris)<!--[![godocs](https://img.shields.io/badge/go-%20docs-488AC7.svg?style=for-the-badge)](https://godoc.org/github.com/kataras/iris)--> [![view examples](https://img.shields.io/badge/learn%20by-examples-0077b3.svg?style=for-the-badge)](https://github.com/kataras/iris/tree/master/_examples) [![chat](https://img.shields.io/gitter/room/iris_go/community.svg?color=blue&logo=gitter&style=for-the-badge)](https://gitter.im/iris_go/community) [![release](https://img.shields.io/badge/release%20-v11.2-0077b3.svg?style=for-the-badge)](https://github.com/kataras/iris/releases)

Iris is a fast, simple yet fully featured and very efficient web framework for Go. It provides a beautifully expressive and easy to use foundation for your next website or API.

Learn what [others say about Iris](https://iris-go.com/testimonials/) and **star** this github repository.

## Learning Iris

<details>
<summary>Qick start</summary>

```sh
# assume the following code in example.go file
$ cat example.go
```

```go
package main

import "github.com/kataras/iris"

func main() {
    app := iris.Default()
    app.Get("/ping", func(ctx iris.Context) {
        ctx.JSON(iris.Map{
            "message": "pong",
        })
    })
    // listen and serve on http://0.0.0.0:8080.
    app.Run(iris.Addr(":8080"))
}
```

```
# run example.go and visit http://0.0.0.0:8080/ping on browser
$ go run example.go
```

> Routing is powered by [muxie](https://github.com/kataras/muxie), the most powerful and fastest trie-based software written in Go.

</details>

Iris contains extensive and thorough **[wiki](https://github.com/kataras/iris/wiki)** making it easy to get started with the framework.

For a more detailed technical documentation you can head over to our [godocs](https://godoc.org/github.com/kataras/iris). And for executable code you can always visit the [_examples](_examples/) repository's subdirectory.

### Do you like to read while traveling?

You can [request](https://bit.ly/iris-req-book) a PDF version and online access of the **E-Book** today and be participated in the development of Iris.

[![https://iris-go.com/images/iris-book-overview.png](https://iris-go.com/images/iris-book-overview.png)](https://bit.ly/iris-req-book)

## Contributing

We'd love to see your contribution to the Iris Web Framework! For more information about contributing to the Iris project please check the [CONTRIBUTING.md](CONTRIBUTING.md) file.

[List of all Contributors](https://github.com/kataras/iris/graphs/contributors)

## Security Vulnerabilities

If you discover a security vulnerability within Iris, please send an e-mail to [iris-go@outlook.com](mailto:iris-go@outlook.com). All security vulnerabilities will be promptly addressed.

## Author

<table>
<tr>
<td>
<img src="https://avatars3.githubusercontent.com/u/22900943?s=460&v=4" width="180"/>

Gerasimos Maropoulos

<p align="center">
<a href = "https://github.com/kataras"><img src = "http://www.iconninja.com/files/241/825/211/round-collaboration-social-github-code-circle-network-icon.svg" width="36" height = "36"/></a>
<a href = "https://twitter.com/MakisMaropoulos"><img src = "https://www.shareicon.net/download/2016/07/06/107115_media.svg" width="36" height="36"/></a>
<a href = "https://www.linkedin.com/in/gerasimos-maropoulos/"><img src = "http://www.iconninja.com/files/863/607/751/network-linkedin-social-connection-circular-circle-media-icon.svg" width="36" height="36"/></a>
</p>
</td>
</tr> 
</table>

## License

The project name "Iris" was insipired by the Greek mythology[*](https://en.wikipedia.org/wiki/Iris_(mythology)).

Iris Web Framework is free and open-source software licensed under the [3-Clause BSD License](LICENSE).

For any questions regarding the license please send [e-mail](mailto:iris-go@outlook.com?subject=Iris%20License).
