# Iris Web Framework <a href="README.md"> <img width="20px" src="https://iris-go.com/images/flag-unitedkingdom.svg?v=10" /></a> <a href="README_ZH.md"><img width="20px" src="https://iris-go.com/images/flag-china.svg?v=10" /></a> <a href="README_GR.md"><img width="20px" src="https://iris-go.com/images/flag-greece.svg?v=10" /></a>

<img align="right" width="169px" src="https://iris-go.com/images/icon.svg?v=a" title="logo created by @merry.dii" />

[![build status](https://img.shields.io/travis/kataras/iris/master.svg?style=flat-square)](https://travis-ci.org/kataras/iris)<!-- [![release](https://img.shields.io/github/release/kataras/iris.svg?style=flat-square)](https://github.com/kataras/iris/releases)--> [![report card](https://img.shields.io/badge/report%20card-a%2B-ff3333.svg?style=flat-square)](http://goreportcard.com/report/kataras/iris) [![vscode-iris](https://img.shields.io/badge/ext%20-vscode-0c77e3.svg?style=flat-square)](https://marketplace.visualstudio.com/items?itemName=kataras2006.iris)<!--[![github closed issues](https://img.shields.io/github/issues-closed-raw/kataras/iris.svg?style=flat-square)](https://github.com/kataras/iris/issues?q=is%3Aissue+is%3Aclosed)--> [![chat](https://img.shields.io/badge/community-%20chat-00BCD4.svg?style=flat-square)](https://kataras.rocket.chat/channel/iris) [![view examples](https://img.shields.io/badge/learn%20by-examples-0077b3.svg?style=flat-square)](_examples/) [![release](https://img.shields.io/badge/release%20-v10.2-0077b3.svg?style=flat-square)](https://github.com/kataras/iris/releases)

Iris - это быстрая, простая, но полнофункциональная и очень эффективная веб-платформа для Go.

Iris предоставляет красиво выразительную и удобную основу для вашего следующего веб-сайта или API.

Наконец, настоящий эквивалент expressjs для языка программирования Go.

Узнайте, что [другие говорят об Iris](#support), и [запустите](https://github.com/kataras/iris/stargazers) этот github-хранилище, чтобы оставаться в курсе последних событий [актуальными](https://facebook.com/iris.framework).

## Сторонники

Спасибо всем, кто поддерживал нас! 🙏 [Поддержать нас](https://opencollective.com/iris#backer)

<a href="https://opencollective.com/iris#backers" target="_blank"><img src="https://opencollective.com/iris/backers.svg?width=890"></a>

```sh
$ cat example.go
```

```go
package main

import "github.com/kataras/iris"

func main() {
    app := iris.New()
    // Load all templates from the "./views" folder
    // where extension is ".html" and parse them
    // using the standard `html/template` package.
    app.RegisterView(iris.HTML("./views", ".html"))

    // Method:    GET
    // Resource:  http://localhost:8080
    app.Get("/", func(ctx iris.Context) {
        // Bind: {{.message}} with "Hello world!"
        ctx.ViewData("message", "Hello world!")
        // Render template file: ./views/hello.html
        ctx.View("hello.html")
    })

    // Method:    GET
    // Resource:  http://localhost:8080/user/42
    //
    // Need to use a custom regexp instead?
    // Easy,
    // just mark the parameter's type to 'string'
    // which accepts anything and make use of
    // its `regexp` macro function, i.e:
    // app.Get("/user/{id:string regexp(^[0-9]+$)}")
    app.Get("/user/{id:long}", func(ctx iris.Context) {
        userID, _ := ctx.Params().GetInt64("id")
        ctx.Writef("User ID: %d", userID)
    })

    // Start the server using a network address.
    app.Run(iris.Addr(":8080"))
}
```

> Чтобы узнать подробнее о типах пути параметров нажмите [здесь](_examples/routing/dynamic-path/main.go#L31)

```html
<!-- file: ./views/hello.html -->
<html>
<head>
    <title>Hello Page</title>
</head>
<body>
    <h1>{{.message}}</h1>
</body>
</html>
```

```sh
$ go run example.go
Now listening on: http://localhost:8080
Application Started. Press CTRL+C to shut down.
_
```

## Установка

Единственное требование [язык программирования Go.](https://golang.org/dl/)

```sh
$ go get -u github.com/kataras/iris
```

Iris использует преимущества функции  [из каталога поставщика](https://docs.google.com/document/d/1Bz5-UB7g2uPBdOx-rw5t9MxJwkfpx90cqG9AFL0JAYo). Вы получаете действительно воспроизводимые конструкции, так как этот метод защищает от восходящего потока переименований и удалений.

[![Iris vs .NET Core(C#) vs Node.js (Express)](https://iris-go.com/images/benchmark-new-gray.png)](_benchmarks/README_UNIX.md)

_Обновлено: [Вторник, 21 ноября 2017 г.](_benchmarks/README_UNIX.md)_

<details>
<summary>Сравнительные тесты сторонних источников по остальным веб-фреймворкам</summary>

![Comparison with other frameworks](https://raw.githubusercontent.com/smallnest/go-web-framework-benchmark/4db507a22c964c9bc9774c5b31afdc199a0fe8b7/benchmark.png)

</details>

## Поддержка

- Файл [HISTORY](HISTORY.md#th-08-february-2018--v1020) - ваш лучший друг, он содержит информацию о последних особенностях и всех изменениях
- Вы случайно обнаружили ошибку? Опубликуйте ее на [Github вопросы](https://github.com/kataras/iris/issues)
- У Вас есть какие-либо вопросы или Вам нужно поговорить с кем-то, кто бы смог решить Вашу проблему в режиме реального времени? Присоединяйтесь к нам в [чате сообщества](https://chat.iris-go.com)
- Заполните наш отчет о пользовательском опыте на основе формы, нажав [здесь](https://docs.google.com/forms/d/e/1FAIpQLSdCxZXPANg_xHWil4kVAdhmh7EBBHQZ_4_xSZVDL-oCC_z5pA/viewform?usp=sf_link) 
- Вам нравится фреймворк? Поделись об этом в Twitter! Люди говорят:

<a href="https://twitter.com/gelnior/status/769100480706379776"> 
    <img src="https://comments.iris-go.com/comment27_mini.png" width="350px">
</a>

<a href="https://twitter.com/MeAlex07/status/822799954188075008"> 
    <img src="https://comments.iris-go.com/comment28_mini.png" width="350px">
</a>

<a href="https://twitter.com/_mgale/status/818591490305761280"> 
    <img src="https://comments.iris-go.com/comment29_mini.png" width="350px">
</a>
<a href="https://twitter.com/VeayoX/status/813273328550973440"> 
    <img src="https://comments.iris-go.com/comment30_mini.png" width="350px">
</a>

<a href="https://twitter.com/pvsukale/status/745328224876408832"> 
    <img src="https://comments.iris-go.com/comment31_mini.png" width="350px">
</a>

<a href="https://twitter.com/blainsmith/status/745338092211560453"> 
    <img src="https://comments.iris-go.com/comment32_mini.png" width="350px">
</a>

<a href="https://twitter.com/tjbyte/status/758287014210867200"> 
    <img src="https://comments.iris-go.com/comment33_mini.png" width="350px">
</a>

<a href="https://twitter.com/tangzero/status/751050577220698112"> 
    <img src="https://comments.iris-go.com/comment34_mini.png" width="350px">
</a>

<a href="https://twitter.com/tjbyte/status/758287244947972096"> 
    <img src="https://comments.iris-go.com/comment33_2_mini.png" width="350px">
</a>

<a href="https://twitter.com/ferarias/status/902468752364773376"> 
    <img src="https://comments.iris-go.com/comment41.png" width="350px">
</a>

<br/><br/>

Для получения дополнительной информации о внесении вклада в проект Iris, пожалуйста, проверьте файл  [CONTRIBUTING.md](CONTRIBUTING.md).

[Список всех участников](https://github.com/kataras/iris/graphs/contributors)

## Учить

Прежде всего, самый правильный способ начать работу с веб-фрэймворк - изучить основы языка программирования и стандартные возможности `http`. Если Ваше веб-приложение представляет собой очень простой персональный проект без производительности и требований к техническому обслуживанию, тогда Вы возможно захотите развиваться просто со стандартным пакетом. После этого следуйте рекомендациям:

- Пройдитесь по **100+1** **[примерам](_examples)** и по некоторым [ стартовым Iris наборам](#iris-starter-kits), которые мы создали для вас
- Прочтите [godocs](https://godoc.org/github.com/kataras/iris) для любых подробностей
- Приготовьте чашечку кофе или чая, что вам больше нравится, и ознакомьтесь с некоторыми  [статьями](#articles), которые мы нашли для вас 

### Стартовые наборы IRIS:

<!-- table form 
| Description | Link |
| -----------|-------------|
| Hasura hub starter project with a ready to deploy golang helloworld webapp with IRIS! | https://hasura.io/hub/project/hasura/hello-golang-iris |
| A basic web app built in Iris for Go |https://github.com/gauravtiwari/go_iris_app |
| A mini social-network created with the awesome Iris💖💖 | https://github.com/iris-contrib/Iris-Mini-Social-Network |
| Iris isomorphic react/hot reloadable/redux/css-modules starter kit | https://github.com/iris-contrib/iris-starter-kit |
| Demo project with react using typescript and Iris | https://github.com/ionutvilie/react-ts |
| Self-hosted Localization Management Platform built with Iris and Angular | https://github.com/iris-contrib/parrot |
| Iris + Docker and Kubernetes | https://github.com/iris-contrib/cloud-native-go |
| Quickstart for Iris with Nanobox | https://guides.nanobox.io/golang/iris/from-scratch |
-->

1. [Основное веб-приложение, созданное в Iris for Go](https://github.com/gauravtiwari/go_iris_app)
2. [Мини-социальная сеть, созданная с удивительным Iris💖💖](https://github.com/iris-contrib/Iris-Mini-Social-Network)
3. [Iris isomorphic react/hot reloadable/redux/css-modules starter kit](https://github.com/iris-contrib/iris-starter-kit)
4. [Демо-проект с реакцией на использованием машинописного текста и Iris](https://github.com/ionutvilie/react-ts)
5. [Самостоятельная платформа управления локализацией, построенная с помощью Iris и Angular](https://github.com/iris-contrib/parrot)
6. [Iris + Docker and Kubernetes](https://github.com/iris-contrib/cloud-native-go)
7. [Быстрый запуск для Iris с Nanobox](https://guides.nanobox.io/golang/iris/from-scratch)
8. [Cтартовый проект Hasura с готовностью применять веб-приложение Golang hello-world с IRIS](https://hasura.io/hub/project/hasura/hello-golang-iris)

> Вы построили что-то подобное? Дайте нам [знать](https://github.com/kataras/iris/pulls)!

### Связующее программное обеспечение

У Iris есть отличный сбор обработчиков[[1]](middleware/)[[2]](https://github.com/iris-contrib/middleware) которые вы можете использовать бок о бок с вашими веб-приложениями. Однако вы не ограничены ими - вы можете использовать стороннее программное обеспечение, совместимое с [net/http](https://golang.org/pkg/net/http/) пакетом, [_examples/convert-handlers](_examples/convert-handlers) покажут вам путь.

Iris, в отличие от других, на 100% совместим со стандартами, и именно поэтому большинство крупных компаний, которые адаптируют Go к своему рабочему процессу, как и очень известная телевизионная сеть США, доверяют Iris; это всегда актуально, и он будет приведен в соответствии с пакетом - `net/http`, который будет модернизирован Автором Go при каждом новом выпуске языка программирования Go навсегда.

### Статьи

* [Приложение Todo MVC с использованием Iris и Vue.js](https://hackernoon.com/a-todo-mvc-application-using-iris-and-vue-js-5019ff870064)
* [Стартовый проект Hasura с готовностью применять веб-приложение Golang hello-world с IRIS](bit.ly/2lmKaAZ)
* [Топ-6 веб-фреймворков для Go на 2017 год
](https://blog.usejournal.com/top-6-web-frameworks-for-go-as-of-2017-23270e059c4b)
* [Iris Go Framework + MongoDB](https://medium.com/go-language/iris-go-framework-mongodb-552e349eab9c)
* [Как создать форму загрузки файла с помощью DropzoneJS и Go](https://hackernoon.com/how-to-build-a-file-upload-form-using-dropzonejs-and-go-8fb9f258a991)
* [Как отображать существующие файлы на сервере с помощью DropzoneJS и Go](https://hackernoon.com/how-to-display-existing-files-on-server-using-dropzonejs-and-go-53e24b57ba19)
* [ Iris, модульная структура сети](https://medium.com/@corebreaker/iris-web-cd684b4685c7)
* [ Go vs .NET Core с точки зрения производительности HTTP](https://medium.com/@kataras/go-vs-net-core-in-terms-of-http-performance-7535a61b67b8)
* [ Iris Go vs .NET Core Kestrel с точки зрения производительности HTTP
](https://hackernoon.com/iris-go-vs-net-core-kestrel-in-terms-of-http-performance-806195dc93d5)
* [Как превратить Android-устройство в веб-сервер](https://twitter.com/ThePracticalDev/status/892022594031017988)
* [Применение приложения Iris Golang на Hasura](https://medium.com/@HasuraHQ/deploy-an-iris-golang-app-with-backend-apis-in-minutes-25a559bf530b)
* [URL-адрес Shortener Service с помощью Go, Iris и Bolt](https://medium.com/@kataras/a-url-shortener-service-using-go-iris-and-bolt-4182f0b00ae7)

### Получить работу

Есть много компаний и стартапов, находящиеся в поисках Go веб-разработчиков с опытом работы с Iris как в качестве требования, которые мы подыскиваем для вас каждый день. Мы публикуем эту информацию на нашей [странице в Facebook](https://www.facebook.com/iris.framework). Ставьте Like, чтобы получите уведомления. Мы уже опубликовали некоторые из них.

### Спонсоры

Спасибо всем нашим спонсорам!  (пожалуйста, попросите вашу компанию также поддержать этот проект с открытым исходным кодом,  [став спонсором](https://opencollective.com/iris#sponsor))

<a href="https://opencollective.com/iris/sponsor/0/website" target="_blank"><img src="https://opencollective.com/iris/sponsor/0/avatar.svg"></a>
<a href="https://opencollective.com/iris/sponsor/1/website" target="_blank"><img src="https://opencollective.com/iris/sponsor/1/avatar.svg"></a>

## Лицензия

Iris лицензируется в соответствии с  [BSD 3-Clause лицензией](LICENSE). Iris - это бесплатное программное обеспечение с открытым исходным кодом на 100%.

По любым вопросам, касающимся лицензии, отправьте письмо на [почту](mailto:kataras2006@hotmail.com?subject=Iris%20License).
