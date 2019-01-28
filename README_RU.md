# Iris Web Framework <a href="README.md"> <img width="20px" src="https://iris-go.com/images/flag-unitedkingdom.svg?v=10" /></a> <a href="README_ZH.md"><img width="20px" src="https://iris-go.com/images/flag-china.svg?v=10" /></a> <a href="README_ID.md"> <img width="20px" src="https://iris-go.com/images/flag-indonesia.svg?v=10" /></a> <a href="README_GR.md"><img width="20px" src="https://iris-go.com/images/flag-greece.svg?v=10" /></a> <a href="README_PT_BR.md"><img width="20px" src="https://iris-go.com/images/flag-pt-br.svg?v=10" /></a> <a href="README_JPN.md"><img width="20px" src="https://iris-go.com/images/flag-japan.svg?v=10" /></a>

<a href="https://iris-go.com"> <img align="right" width="169px" src="https://iris-go.com/images/icon.svg?v=a" title="logo created by @merry.dii" /> </a>

[![build status](https://img.shields.io/travis/kataras/iris/master.svg?style=flat-square)](https://travis-ci.org/kataras/iris)<!-- [![release](https://img.shields.io/github/release/kataras/iris.svg?style=flat-square)](https://github.com/kataras/iris/releases)--> [![report card](https://img.shields.io/badge/report%20card-a%2B-ff3333.svg?style=flat-square)](http://goreportcard.com/report/kataras/iris) [![vscode-iris](https://img.shields.io/badge/ext%20-vscode-0c77e3.svg?style=flat-square)](https://marketplace.visualstudio.com/items?itemName=kataras2006.iris)<!--[![github closed issues](https://img.shields.io/github/issues-closed-raw/kataras/iris.svg?style=flat-square)](https://github.com/kataras/iris/issues?q=is%3Aissue+is%3Aclosed)--> [![chat](https://img.shields.io/badge/community-%20chat-00BCD4.svg?style=flat-square)](https://kataras.rocket.chat/channel/iris) [![view examples](https://img.shields.io/badge/learn%20by-examples-0077b3.svg?style=flat-square)](https://github.com/kataras/iris/tree/master/_examples/routing) [![release](https://img.shields.io/badge/release%20-v11.1-0077b3.svg?style=flat-square)](https://github.com/kataras/iris/releases)

Iris - это быстрая, простая, но полнофункциональная и очень эффективная веб-платформа для Go.

Iris предоставляет красиво выразительную и удобную основу для вашего следующего веб-сайта или API.

Наконец, настоящий эквивалент expressjs для языка программирования Go.

Узнайте, что [другие говорят об Iris](#support), и [запустите](https://github.com/kataras/iris/stargazers) этот github-хранилище, чтобы оставаться в курсе последних событий [актуальными](https://facebook.com/iris.framework).

## Сторонники

Спасибо всем, кто поддерживал нас! 🙏 [Поддержать нас](https://iris-go.com/donate)

<a href="https://iris-go.com/donate" target="_blank"><img src="https://iris-go.com/backers.svg?v=2"/></a>

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

- Файл [HISTORY](HISTORY.md#fr-11-january-2019--v1111) - ваш лучший друг, он содержит информацию о последних особенностях и всех изменениях
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

1. [snowlyg/IrisApiProject: Iris + gorm + jwt + sqlite3](https://github.com/snowlyg/IrisApiProject) **NEW-Chinese**
2. [yz124/superstar: Iris + xorm to implement the star library](https://github.com/yz124/superstar) **NEW-Chinese**
3. [jebzmos4/Iris-golang: A basic CRUD API in golang with Iris](https://github.com/jebzmos4/Iris-golang)
4. [gauravtiwari/go_iris_app: A basic web app built in Iris for Go](https://github.com/gauravtiwari/go_iris_app)
5. [A mini social-network created with the awesome Iris💖💖](https://github.com/iris-contrib/Iris-Mini-Social-Network)
6. [Iris isomorphic react/hot reloadable/redux/css-modules starter kit](https://github.com/iris-contrib/iris-starter-kit)
7. [ionutvilie/react-ts: Demo project with react using typescript and Iris](https://github.com/ionutvilie/react-ts)
8. [Self-hosted Localization Management Platform built with Iris and Angular](https://github.com/iris-contrib/parrot)
9. [Iris + Docker and Kubernetes](https://github.com/iris-contrib/cloud-native-go)
10. [nanobox.io: Quickstart for Iris with Nanobox](https://guides.nanobox.io/golang/iris/from-scratch)
11. [hasura.io: A Hasura starter project with a ready to deploy Golang hello-world web app with IRIS](https://hasura.io/hub/project/hasura/hello-golang-iris)

> Вы построили что-то подобное? Дайте нам [знать](https://github.com/kataras/iris/pulls)!

### Связующее программное обеспечение

У Iris есть отличный сбор обработчиков[[1]](middleware/)[[2]](https://github.com/iris-contrib/middleware) которые вы можете использовать бок о бок с вашими веб-приложениями. Однако вы не ограничены ими - вы можете использовать стороннее программное обеспечение, совместимое с [net/http](https://golang.org/pkg/net/http/) пакетом, [_examples/convert-handlers](_examples/convert-handlers) покажут вам путь.

Iris, в отличие от других, на 100% совместим со стандартами, и именно поэтому большинство крупных компаний, которые адаптируют Go к своему рабочему процессу, как и очень известная телевизионная сеть США, доверяют Iris; это всегда актуально, и он будет приведен в соответствии с пакетом - `net/http`, который будет модернизирован Автором Go при каждом новом выпуске языка программирования Go навсегда.

### Статьи

* [CRUD REST API in Iris (a framework for golang)](https://medium.com/@jebzmos4/crud-rest-api-in-iris-a-framework-for-golang-a5d33652401e)
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

### Video Courses

* [Daily Coding - Web Framework Golang: Iris Framework]( https://www.youtube.com/watch?v=BmOLFQ29J3s) by WarnabiruTV, source: youtube, cost: **FREE**
* [Tutorial Golang MVC dengan Iris Framework & Mongo DB](https://www.youtube.com/watch?v=uXiNYhJqh2I&list=PLMrwI6jIZn-1tzskocnh1pptKhVmWdcbS) (19 parts so far) by Musobar Media, source: youtube, cost: **FREE**
* [Go/Golang 27 - Iris framework : Routage de base](https://www.youtube.com/watch?v=rQxRoN6ub78) by stephgdesign, source: youtube, cost: **FREE**
* [Go/Golang 28 - Iris framework : Templating](https://www.youtube.com/watch?v=nOKYV073S2Y) by stephgdesignn, source: youtube, cost: **FREE**
* [Go/Golang 29 - Iris framework : Paramètres](https://www.youtube.com/watch?v=K2FsprfXs1E) by stephgdesign, source: youtube, cost: **FREE**
* [Go/Golang 30 - Iris framework : Les middelwares](https://www.youtube.com/watch?v=BLPy1So6bhE) by stephgdesign, source: youtube, cost: **FREE**
* [Go/Golang 31 - Iris framework : Les sessions](https://www.youtube.com/watch?v=RnBwUrwgEZ8) by stephgdesign, source: youtube, cost: **FREE**

### Получить работу

Есть много компаний и стартапов, находящиеся в поисках Go веб-разработчиков с опытом работы с Iris как в качестве требования, которые мы подыскиваем для вас каждый день. Мы публикуем эту информацию на нашей [странице в Facebook](https://www.facebook.com/iris.framework). Ставьте Like, чтобы получите уведомления. Мы уже опубликовали некоторые из них.

## Лицензия

Iris лицензируется в соответствии с  [BSD 3-Clause лицензией](LICENSE). Iris - это бесплатное программное обеспечение с открытым исходным кодом на 100%.

По любым вопросам, касающимся лицензии, отправьте письмо на [почту](mailto:kataras2006@hotmail.com?subject=Iris%20License).
