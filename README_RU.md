# Iris Web Framework

[![build status](https://img.shields.io/github/actions/workflow/status/kataras/iris/ci.yml?branch=main&style=for-the-badge)](https://github.com/kataras/iris/actions/workflows/ci.yml) [![FOSSA Status](https://img.shields.io/badge/LICENSE%20SCAN-PASSING❤️-CD2956?style=for-the-badge&logo=fossa)](https://app.fossa.io/projects/git%2Bgithub.com%2Fkataras%2Firis?ref=badge_shield)<!--[![report card](https://img.shields.io/badge/report%20card-a%2B-ff3333.svg?style=for-the-badge)](https://goreportcard.com/report/github.com/kataras/iris)--><!--[![godocs](https://img.shields.io/badge/go-%20docs-488AC7.svg?style=for-the-badge)](https://pkg.go.dev/github.com/kataras/iris/v12@v12.2.10)--> [![view examples](https://img.shields.io/badge/learn%20by-examples-0C8EC5.svg?style=for-the-badge&logo=go)](https://github.com/kataras/iris/tree/main/_examples) [![chat](https://img.shields.io/gitter/room/iris_go/community.svg?color=7E18DD&logo=gitter&style=for-the-badge)](https://gitter.im/iris_go/community)<!--[![donate on PayPal](https://img.shields.io/badge/support-PayPal-blue.svg?style=for-the-badge)](https://iris-go.com/donate)--><!-- [![release](https://img.shields.io/badge/release%20-v12.0-0077b3.svg?style=for-the-badge)](https://github.com/kataras/iris/releases) -->
Iris — это быстрый, простой, но полнофункциональный и эффективный веб-фреймворк для Go. Он обеспечивает красивую, выразительную и простую в использовании основу для вашего следующего веб-сайта или API.

Узнайте, что [говорят другие люди об Iris](https://iris-go.com/testimonials/) и поставьте **[звёздочку](https://github.com/kataras/iris/stargazers)** этому проекту с открытым исходным кодом, чтобы поддержать его потенциал.

[![](https://media.giphy.com/media/j5WLmtvwn98VPrm7li/giphy.gif)](https://iris-go.com/testimonials/)

[![Benchmarks: Apr 2, 2020 at 12:13pm (UTC)](https://iris-go.com/images/benchmarks.svg)](https://github.com/kataras/server-benchmarks)

## Изучение Iris

<details>
<summary>Быстрый старт</summary>

```sh
# например, код в файле example.go будет таким:
$ cat example.go
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
# запустите example.go и перейдите в браузер
# по адресу http://localhost:8080/ping
$ go run example.go
```

> Система роутинга запросов работает на [muxie](https://github.com/kataras/muxie), мощное и быстрое trie-based ПО, написанное на Go.

</details>

У Iris есть исчерпывающий и тщательный **[wiki](https://www.iris-go.com/#ebookDonateForm)**, который позволит вам быстрее начать работу с фреймворком.

<!-- ![](https://media.giphy.com/media/Ur8iqy9FQfmPuyQpgy/giphy.gif) -->

Для получения более подробной технической документации вы можете обратиться к нашему [godoc](https://pkg.go.dev/github.com/kataras/iris/v12@v12.2.10). А для живых примеров кода — вы всегда можете посетить [\_examples](_examples/) в поддиректории этого репозитория.

### Вы любите читать во время путешествий?

<a href="https://iris-go.com/#book"> <img alt="Book cover" src="https://iris-go.com/images/iris-book-cover-sm.jpg?v=12" /> </a>

<!-- [![follow author](https://img.shields.io/twitter/follow/makismaropoulos.svg?style=for-the-badge)](https://twitter.com/intent/follow?screen_name=makismaropoulos) -->

Вы можете [запросить](https://www.iris-go.com/#ebookDonateForm) PDF версию и онлайн-доступ к **E-Book** сегодня и принять участие в разработке Iris.

## Содействие

Мы будем рады видеть ваш вклад в веб-фреймворк Iris! Для получения дополнительной информации о содействии проекту Iris, пожалуйста, проверьте файл [CONTRIBUTING.md](CONTRIBUTING.md).

[Список всех участников](https://github.com/kataras/iris/graphs/contributors)

## Уязвимость безопасности

Если вы обнаружите уязвимость безопасности в Iris, отправьте электронное письмо по адресу [iris-go@outlook.com](mailto:iris-go@outlook.com). Все уязвимости безопасности будут оперативно устранены.

## Лицензия

Название проекта «Iris» было вдохновлено греческой мифологией.

Веб-фреймворк Iris — это ПО с открытым исходным кодом под лицензией [3-Clause BSD License](LICENSE).

## Накопление звёзд со временем

[![Stargazers over time](https://starchart.cc/kataras/iris.svg)](https://starchart.cc/kataras/iris)
