# Iris Web Framework

[![build status](https://img.shields.io/github/actions/workflow/status/kataras/iris/ci.yml?branch=main&style=for-the-badge)](https://github.com/kataras/iris/actions/workflows/ci.yml) [![FOSSA Status](https://img.shields.io/badge/LICENSE%20SCAN-PASSING❤️-CD2956?style=for-the-badge&logo=fossa)](https://app.fossa.io/projects/git%2Bgithub.com%2Fkataras%2Firis?ref=badge_shield)<!--[![report card](https://img.shields.io/badge/report%20card-a%2B-ff3333.svg?style=for-the-badge)](https://goreportcard.com/report/github.com/kataras/iris)--><!--[![godocs](https://img.shields.io/badge/go-%20docs-488AC7.svg?style=for-the-badge)](https://pkg.go.dev/github.com/kataras/iris/v12@v12.2.10)--> [![view examples](https://img.shields.io/badge/learn%20by-examples-0C8EC5.svg?style=for-the-badge&logo=go)](https://github.com/kataras/iris/tree/main/_examples) [![chat](https://img.shields.io/gitter/room/iris_go/community.svg?color=7E18DD&logo=gitter&style=for-the-badge)](https://gitter.im/iris_go/community)<!--[![donate on PayPal](https://img.shields.io/badge/support-PayPal-blue.svg?style=for-the-badge)](https://iris-go.com/donate)--><!-- [![release](https://img.shields.io/badge/release%20-v12.0-0077b3.svg?style=for-the-badge)](https://github.com/kataras/iris/releases) -->

<a href="https://iris-go.com"> <img align="right" src="https://iris-go.com/images/logo-w169.png"></a>

Iris est un framework open-source pour Go à la fois simple, rapide et pourvu de nombreuses fonctionnalités.

Il fournit des moyens simples et élégants de construire les bases et fonctionnalités de votre site, application backend ou API Rest.

Lisez [ce que les développeurs pensent d'Iris](https://iris-go.com/testimonials/) et si l'envie vous prend **[étoilez](https://github.com/kataras/iris/stargazers)** le projet pour faire monter son potentiel.

[![](https://media.giphy.com/media/j5WLmtvwn98VPrm7li/giphy.gif)](https://iris-go.com/testimonials/)

[![Benchmarks: Apr 2, 2020 at 12:13pm (UTC)](https://iris-go.com/images/benchmarks.svg)](https://github.com/kataras/server-benchmarks)

## 📖 Démarrer avec Iris

<details>
<summary>Un simple Hello World</summary>

```sh
# https://www.iris-go.com/#ebookDonateForm
$ go get github.com/kataras/iris/v12@latest
# assume the following code in example.go file
$ cat example.go
```

```go
package main

import "github.com/kataras/iris/v12"

func main() {
    app := iris.New()
    app.Get("/ping", func(ctx iris.Context) {
        ctx.JSON(iris.Map{
            "message": "pong",
        })
    })

    app.Listen(":8080")  // port d'écoute
}
```

```sh
# compile et execute example.go
$ go run example.go
# maintenant visitez http://localhost:8080/ping
```

> Le routing est géré par [muxie](https://github.com/kataras/muxie), la librairie Go la plus rapide et complète.

</details>

Iris possède un **[wiki](https://www.iris-go.com/#ebookDonateForm)** complet et précis qui vous permettra d'implémenter ses fonctionnalités rapidement et facilement.

<!-- ![](https://media.giphy.com/media/Ur8iqy9FQfmPuyQpgy/giphy.gif) -->

Pour une documentation encore plus complète vous pouvez visiter notre [godocs](https://pkg.go.dev/github.com/kataras/iris/v12@v12.2.10) (en Anglais). Et vous trouverez du code executable dans le dossier [\_examples](_examples/).

### Vous préférez une version PDF?

<a href="https://iris-go.com/#book"> <img alt="Book cover" src="https://iris-go.com/images/iris-book-cover-sm.jpg?v=12"/> </a>

[![follow author](https://img.shields.io/twitter/follow/makismaropoulos.svg?style=for-the-badge)](https://twitter.com/intent/follow?screen_name=makismaropoulos)

Vous pouvez [demander](https://www.iris-go.com/#ebookDonateForm) une version **E-Book** (en Anglais) de la documentation et contribuer au développement d'Iris.

## 🙌 Contribuer

Toute contribution à Iris est la bienvenue ! Pour plus d'informations sur la contribution au projet référez-vous au fichier [CONTRIBUTING.md](CONTRIBUTING.md).

[Liste des contributeurs](https://github.com/kataras/iris/graphs/contributors)

## 🛡 Sécurité et vulnérabilités

Si vous trouvez une vulnérabilité dans Iris, envoyez un e-mail à [iris-go@outlook.com](mailto:iris-go@outlook.com). Toute vulnérabilité sera corrigée aussi rapidement que possible.

## 📝 Licence

Le projet est sous licence [licence BSD 3](LICENSE), tout comme le langage Go lui même.

Le nom "Iris" est inspiré de la mythologie Grecque.
<!-- ## Stargazers over time

[![Stargazers over time](https://starchart.cc/kataras/iris.svg)](https://starchart.cc/kataras/iris) -->
