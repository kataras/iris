<!-- # Iris Web Framework <a href="README_ZH.md"> <img width="20px" src="https://iris-go.com/images/flag-china.svg?v=10" /></a> <a href="README_RU.md"><img width="20px" src="https://iris-go.com/images/flag-russia.svg?v=10" /></a> <a href="README_ID.md"> <img width="20px" src="https://iris-go.com/images/flag-indonesia.svg?v=10" /></a> <a href="README_GR.md"><img width="20px" src="https://iris-go.com/images/flag-greece.svg?v=10" /></a> <a href="README_PT_BR.md"><img width="20px" src="https://iris-go.com/images/flag-pt-br.svg?v=10" /></a> <a href="README_JPN.md"><img width="20px" src="https://iris-go.com/images/flag-japan.svg?v=10" /></a> -->

# Iris <a href="README.md"> <img width="20px" src="https://iris-go.com/images/flag-unitedkingdom.svg?v=10" /></a> <a href="README_ZH.md"><img width="20px" src="https://iris-go.com/images/flag-china.svg?v=10" /></a> <a href="README_GR.md"><img width="20px" src="https://iris-go.com/images/flag-greece.svg?v=10" /></a> <a href="README_KO.md"><img width="20px" src="https://iris-go.com/images/flag-south-korea.svg" /></a> <a href="README_FA.md"><img width="20px" src="https://iris-go.com/images/flag-iran.svg" /></a> <a href="README_RU.md"><img width="20px" src="https://iris-go.com/images/flag-russia.svg" /></a>

[![build status](https://img.shields.io/travis/kataras/iris/master.svg?style=for-the-badge&logo=travis)](https://travis-ci.org/kataras/iris) [![report card](https://img.shields.io/badge/report%20card-a%2B-ff3333.svg?style=for-the-badge)](https://goreportcard.com/report/github.com/kataras/iris)<!--[![godocs](https://img.shields.io/badge/go-%20docs-488AC7.svg?style=for-the-badge)](https://godoc.org/github.com/kataras/iris)--> [![view examples](https://img.shields.io/badge/learn%20by-examples-0077b3.svg?style=for-the-badge)](https://github.com/kataras/iris/tree/master/_examples) [![chat](https://img.shields.io/gitter/room/iris_go/community.svg?color=blue&logo=gitter&style=for-the-badge)](https://gitter.im/iris_go/community) [![release](https://img.shields.io/badge/release%20-v11.2-0077b3.svg?style=for-the-badge)](https://github.com/kataras/iris/releases)

Iris es un framework web rápido, simple pero con muchas funcionalidades y muy eficiente para Go. Proporciona una base bellamente expresiva y fácil de usar para su próximo sitio web o API.

Descubra lo que [otros dicen sobre Iris](https://iris-go.com/testimonials/) y **siga** :star: este repositorio github.

[![](https://media.giphy.com/media/j5WLmtvwn98VPrm7li/giphy.gif)](https://iris-go.com/testimonials/)

## Aprende Iris

<details>
<summary>Inicio rapido</summary>

```sh
# agrega el siguiente código en el archivo ejemplo.go
$ cat ejemplo.go
```

```go
package main

import "github.com/kataras/iris/v12"

func main() {
    app := iris.Default()
    app.Get("/ping", func(ctx iris.Context) {
        ctx.JSON(iris.Map{
            "message": "pong",
        })
    })

    app.Run(iris.Addr(":8080"))
}
```

```sh
# ejecuta ejemplo.go y
# visita http://localhost:8080/ping en el navegador
$ go run ejemplo.go
```

> El enrutamiento es impulsado por [muxie](https://github.com/kataras/muxie), el software basado en trie más potente y rápido escrito en Go.

</details>

Iris contiene un extenso y completo **[wiki](https://github.com/kataras/iris/wiki)** que facilita comenzar con el framework.

Para obtener una documentación técnica más detallada, puede dirigirse a nuestros [godocs](https://godoc.org/github.com/kataras/iris). Y para código ejecutable siempre puede visitar el subdirectorio del repositorio [\_examples](_examples/).

### ¿Te gusta leer mientras viajas?

<a href="https://bit.ly/iris-req-book"> <img alt="Book cover" src="https://iris-go.com/images/iris-book-cover-sm.jpg" width="200" /> </a>

[![follow author](https://img.shields.io/twitter/follow/makismaropoulos.svg?style=for-the-badge)](https://twitter.com/intent/follow?screen_name=makismaropoulos)

Puedes [solicitar](https://bit.ly/iris-req-book) una versión en PDF y acceso en línea del **E-Book** hoy y participar en el desarrollo de Iris.

## Contribuir

¡Nos encantaría ver su contribución al Framework Web Iris! Para obtener más información sobre cómo contribuir al proyecto Iris, consulte el archivo [CONTRIBUTING.md](CONTRIBUTING).

[Lista de todos los contribuyentes](https://github.com/kataras/iris/graphs/contributors)

## Vulnerabilidades de seguridad

Si descubres una vulnerabilidad de seguridad dentro de Iris, envíe un correo electrónico a [iris-go@outlook.com](mailto:iris-go@outlook.com). Todas las vulnerabilidades de seguridad serán tratadas de inmediato.

## Licencia

El nombre del proyecto "Iris" se inspiró en la mitología griega.

El Web Framework Iris es un software gratuito y de código abierto con licencia bajo la [Licencia BSD 3 cláusulas](LICENSE).
