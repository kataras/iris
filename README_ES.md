# Iris Web Framework

[![build status](https://img.shields.io/github/actions/workflow/status/kataras/iris/ci.yml?branch=main&style=for-the-badge)](https://github.com/kataras/iris/actions/workflows/ci.yml) [![FOSSA Status](https://img.shields.io/badge/LICENSE%20SCAN-PASSING❤️-CD2956?style=for-the-badge&logo=fossa)](https://app.fossa.io/projects/git%2Bgithub.com%2Fkataras%2Firis?ref=badge_shield)<!--[![report card](https://img.shields.io/badge/report%20card-a%2B-ff3333.svg?style=for-the-badge)](https://goreportcard.com/report/github.com/kataras/iris)--><!--[![godocs](https://img.shields.io/badge/go-%20docs-488AC7.svg?style=for-the-badge)](https://pkg.go.dev/github.com/kataras/iris/v12@v12.2.11)--> [![view examples](https://img.shields.io/badge/learn%20by-examples-0C8EC5.svg?style=for-the-badge&logo=go)](https://github.com/kataras/iris/tree/main/_examples) [![chat](https://img.shields.io/gitter/room/iris_go/community.svg?color=7E18DD&logo=gitter&style=for-the-badge)](https://gitter.im/iris_go/community)<!--[![donate on PayPal](https://img.shields.io/badge/support-PayPal-blue.svg?style=for-the-badge)](https://iris-go.com/donate)--><!-- [![release](https://img.shields.io/badge/release%20-v12.0-0077b3.svg?style=for-the-badge)](https://github.com/kataras/iris/releases) -->

Iris es un framework web rápido, simple pero con muchas funcionalidades y muy eficiente para Go. Proporciona una base bellamente expresiva y fácil de usar para su próximo sitio web o API.

Descubra lo que [otros dicen sobre Iris](https://iris-go.com/testimonials/) y **siga** :star: este repositorio github.

[![](https://media.giphy.com/media/j5WLmtvwn98VPrm7li/giphy.gif)](https://iris-go.com/testimonials/)

[![Benchmarks: Apr 2, 2020 at 12:13pm (UTC)](https://iris-go.com/images/benchmarks.svg)](https://github.com/kataras/server-benchmarks)

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

    app.Listen(":8080")
}
```

```sh
# ejecuta ejemplo.go y
# visita http://localhost:8080/ping en el navegador
$ go run ejemplo.go
```

> El enrutamiento es impulsado por [muxie](https://github.com/kataras/muxie), el software basado en trie más potente y rápido escrito en Go.

</details>

Iris contiene un extenso y completo **[wiki](https://www.iris-go.com/#ebookDonateForm)** que facilita comenzar con el framework.

Para obtener una documentación técnica más detallada, puede dirigirse a nuestros [godocs](https://pkg.go.dev/github.com/kataras/iris/v12@v12.2.11). Y para código ejecutable siempre puede visitar el subdirectorio del repositorio [\_examples](_examples/).

### ¿Te gusta leer mientras viajas?

<a href="https://iris-go.com/#book"> <img alt="Book cover" src="https://iris-go.com/images/iris-book-cover-sm.jpg?v=12" /> </a>

[![follow author](https://img.shields.io/twitter/follow/makismaropoulos.svg?style=for-the-badge)](https://twitter.com/intent/follow?screen_name=makismaropoulos)

Puedes [solicitar](https://www.iris-go.com/#ebookDonateForm) una versión en PDF y acceso en línea del **E-Book** hoy y participar en el desarrollo de Iris.

## Contribuir

¡Nos encantaría ver su contribución al Framework Web Iris! Para obtener más información sobre cómo contribuir al proyecto Iris, consulte el archivo [CONTRIBUTING.md](CONTRIBUTING).

[Lista de todos los contribuyentes](https://github.com/kataras/iris/graphs/contributors)

## Vulnerabilidades de seguridad

Si descubres una vulnerabilidad de seguridad dentro de Iris, envíe un correo electrónico a [iris-go@outlook.com](mailto:iris-go@outlook.com). Todas las vulnerabilidades de seguridad serán tratadas de inmediato.

## Licencia

El nombre del proyecto "Iris" se inspiró en la mitología griega.

El Web Framework Iris es un software gratuito y de código abierto con licencia bajo la [Licencia BSD 3 cláusulas](LICENSE).
