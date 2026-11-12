# <a href="https://iris-go.com"><img src="https://iris-go.com/images/logo-new-lq-45.png"></a> Iris Web Framework <a href="README_GR.md"><img width="20px" src="https://iris-go.com/images/flag-greece.svg" /> <a href="README_JA.md"><img width="20px" height="20px" src="https://iris-go.com/images/flag-japan.svg" /></a> </a> <a href="README_FR.md"><img width="20px" src="https://iris-go.com/images/flag-france.svg" /></a> <a href="README_ZH_HANT.md"><img width="20px" src="https://iris-go.com/images/flag-taiwan.svg" /></a> <a href="README_ZH_HANS.md"><img width="20px" src="https://iris-go.com/images/flag-china.svg" /></a> <a href="README_ES.md"><img width="20px" src="https://iris-go.com/images/flag-spain.png" /></a> <a href="README_FA.md"><img width="20px" src="https://iris-go.com/images/flag-iran.svg" /></a> <a href="README_RU.md"><img width="20px" src="https://iris-go.com/images/flag-russia.svg" /></a> <a href="README_KO.md"><img width="20px" src="https://iris-go.com/images/flag-south-korea.svg?v=12" /></a> <a href="README_PT_BR.md"><img width="20px" height="20px" src="https://iris-go.com/images/flag-brazil.svg" /></a> <a href="README_VN.md"><img width="20px" height="20px" src="https://iris-go.com/images/flag-vietnam.svg" /></a>

[![build status](https://img.shields.io/github/actions/workflow/status/kataras/iris/ci.yml?branch=main&style=for-the-badge)](https://github.com/kataras/iris/actions/workflows/ci.yml) [![release](https://img.shields.io/badge/release%20-v14.0-0077b3.svg?style=for-the-badge)](https://github.com/kataras/iris/releases) [![view examples](https://img.shields.io/badge/examples%20-59-a83adf.svg?style=for-the-badge&logo=go)](_examples/README.md) [![godocs](https://img.shields.io/badge/go-%20docs-488AC7.svg?style=for-the-badge)](https://pkg.go.dev/github.com/kataras/iris/v14@v14.0.0) [![chat](https://img.shields.io/gitter/room/iris_go/community.svg?color=cc2b5e&logo=gitter&style=for-the-badge)](https://gitter.im/iris_go/community) [![donate](https://img.shields.io/badge/support-Iris-blue.svg?style=for-the-badge&logo=paypal)](https://iris-go.com/donate)

Iris — это быстрый веб-фреймворк для Go с «батарейками в комплекте». Он даёт вашему следующему API или сайту выразительную маршрутизацию с типизированными параметрами пути, единое место для обработки ошибок, внедрение зависимостей, которое сохраняет код тестируемым, и производственные настройки по умолчанию, безопасные из коробки. <!-- human-writing: ok, тире вместо опущенной связки -->

**Версия 14** — крупнейший релиз за всю историю проекта. Фреймворк был перестроен вокруг дженериков Go: билдер приложения с проверками на этапе компиляции, типизированные хелперы запросов, которые выполняют валидацию за вас, контроллеры, разрешаемые контейнером зависимостей, и конвейер обработки ошибок, который настраивается один раз. В [HISTORY.md](HISTORY.md) записано всё новое; [Руководство по миграции](MIGRATION.md) переводит приложения с v12 на новую версию за один подход. <!-- human-writing: ok, тире вместо опущенной связки -->

> 🤖 **Спросите ИИ об Iris.** Бесплатная [Code Wiki для Iris](https://codewiki.google/github.com/kataras/iris) от Google генерируется из этого репозитория и остаётся синхронизированной с ним, коммит за коммитом. Изучайте архитектуру по диаграммам или задавайте встроенному чату вопросы о кодовой базе обычным языком, и получайте ответы, основанные на реальном исходном коде.

Узнайте, что [другие говорят об Iris](https://www.iris-go.com/#review), и поставьте **[звезду](https://github.com/kataras/iris/stargazers)** этому open-source проекту, чтобы поддержать его развитие. Интересует скорость? Смотрите независимые [бенчмарки серверов](https://github.com/kataras/server-benchmarks).

[![](https://iris-go.com/images/reviews.gif)](https://iris-go.com/testimonials/)

```go
package main

import "github.com/kataras/iris/v14"

func main() {
    app := iris.New()

    app.Get("/", func(ctx iris.Context) {
        ctx.JSON(iris.Map{"message": "Hello, World!"})
    })

    app.Get("/greet/{name:string}", func(ctx iris.Context) {
        ctx.WriteString("Hello, " + ctx.Params().GetString("name"))
    })

    app.Listen(":8080")
}
```

<details><summary>Производственный стек: в одном выражении</summary>

Реальные приложения используют `iris.NewBuilder`: цепочку с проверками на этапе
компиляции, которая подключает CORS, сжатие, логи доступа, health-эндпоинт,
вашу карту ошибок, ваши сервисы и ваши группы API. Неверный порядок шагов
компилятор отклоняет.

```go
iris.NewBuilder().
    Prefix("/api").
    AllowOrigin("*").
    Compression(true).
    LogRequests(true).
    Health(true, "production", "kataras").
    Errors(errors.NewOptions().
        MapErrors(errors.NotFound, catalog.ErrNotFound)).
    Services(catalog.NewRepository, catalog.NewService).
    RouterMiddlewares().
    Middlewares().
    API("/products", api.NewProductsAPI).
    Build().
    Listen(":8080")
```

Обработчики остаются длиной в шесть строк, потому что декодирование,
валидация и отображение ошибок происходят в одном настроенном месте:

```go
func (api *ProductsAPI) create(ctx iris.Context) {
    input, ok := ctx.BindJSON[catalog.ProductInput]()
    if !ok {
        return // 400 already written: malformed JSON or failed Validate.
    }

    id, err := api.svc.Create(ctx, input)
    ctx.Created(id, err) // 201, or the centrally mapped error.
}
```

[Вводный учебник](TUTORIAL.md) в одном документе строит на этом стеке
полноценный сервис, покрытый тестами.

</details>


Как однажды сказал один [Go-разработчик](https://twitter.com/dkuye/status/1532087942696554497), **Iris прикрывает вас со всех сторон и уверенно держится на протяжении многих лет**.

Что вы получаете с Iris v14:

* Маршрутизация с типизированными параметрами пути (`{id:uuid}`, `{n:int}`, `{addr:email}` и другими), пользовательские макросы и ограничения. Некорректный ввод отклоняется ещё до запуска вашего обработчика.
* `iris.NewBuilder`: CORS, сжатие, логирование доступа, health-эндпоинт, маппинг ошибок, сервисы и группы API, собранные в одну цепочку с проверками компилятора.
* Внедрение зависимостей: конструкторы объявляют, что им нужно, а контейнер это предоставляет. Хуки `Initializer` при старте, автоматическое закрытие ресурсов при остановке.
* Хелперы `rest` и одна центральная карта ошибок: тонкие обработчики, канонические машиночитаемые ошибки, внутренние детали никогда не утекают к клиентам.
* Полезные нагрузки запросов, которые валидируют сами себя: реализуйте `Validate() error`, и `ctx.BindJSON` выполнит его после декодирования.
* Тридцать один встроенный middleware: accesslog, API-ключи, basicauth, bodylimit, cache, CORS, hCaptcha и reCAPTCHA, JWT, monitor, pprof, ограничение частоты запросов, recovery, request ID, rewrite, сессии, версионирование и другие.
* Аутентификация любого масштаба: Basic Auth в одну строку, API-ключи, обобщённый JWT SDK, полноценный identity-сервер и OAuth2-провайдер.
* MVC-контроллеры на основе дженериков Go.
* WebSocket, обслуживание gRPC и версионирование API.
* Движки представлений: HTML, Django, Handlebars, Pug и другие через iris-contrib/views.
* Локализация с плюрализацией и определением языка, а также sitemap.
* Сессии с безопасными настройками cookie по умолчанию (`Secure`, `SameSite=Lax` из коробки).
* Богатые ответы: JSON, JSONP, XML, YAML, Markdown, Protocol Buffers, MessagePack, согласование содержимого, стриминг и Server-Sent Events.
* AI-бэкенды: пакет `ai` рендерит и ретранслирует по HTTP потоковый вывод моделей (текстовые дельты, трассировки рассуждений, фрагменты вызовов инструментов, статистика использования). Он не привязан к провайдеру и не включает ни одного вендорского SDK.
* HTTP-клиент, пакет `client`, для вызова других сервисов по тем же правилам JSON, что и на стороне сервера.
* Сжатие ответов: gzip, deflate, brotli, snappy, s2, zstd.
* TLS с Let's Encrypt и встроенное туннелирование ngrok (opt-in: `import _ "github.com/kataras/iris/v14/core/server/ngrok"`), чтобы за секунды вывести dev-приложение в интернет.
* HTTP-тестирование в памяти: тесты всего приложения без портов, без сети и без задержек.
* Полноценная книга, которую можно бесплатно читать онлайн и скачать, практический учебник и живая AI-вики (см. раздел «Изучение Iris» ниже).
* И самое главное: быстрые ответы и поддержка, с первого дня и по сей день.

## 📖 Изучение Iris

> Впервые знакомитесь с Iris? Начните с [вводного учебника](TUTORIAL.md),
> одного документа, который проведёт вас от `go mod init` до протестированного
> REST API производственного уровня. Переходите с v12? Смотрите
> [Руководство по миграции](MIGRATION.md). Возникли вопросы? Спросите
> бесплатную **живую AI-вики** по адресу
> [codewiki.google/github.com/kataras/iris](https://codewiki.google/github.com/kataras/iris).
> Она генерируется Google из исходного кода Iris и поддерживается с ним в
> синхронизации, а встроенный чат отвечает на вопросы о кодовой базе.

### Установка

Единственное требование — [язык программирования Go](https://go.dev/dl/) версии 1.27 или новее. <!-- human-writing: ok, тире вместо опущенной связки -->

#### Создайте новый проект

```sh
$ mkdir myapp
$ cd myapp
$ go mod init myapp
$ go get github.com/kataras/iris/v14@latest # or @v14.0.0
```

<details><summary>Установка в существующий проект</summary>

```sh
$ cd myapp
$ go get github.com/kataras/iris/v14@latest
```

**Запуск**

```sh
$ go mod tidy
$ go run .
```

</details>

![](https://www.iris-go.com/images/gifs/install-create-iris.gif)

Iris сопровождается обширной и подробной **[документацией](https://www.iris-go.com/docs)**, благодаря которой начать работу с фреймворком очень легко.

За более детальной технической документацией обращайтесь к нашим [godocs](https://pkg.go.dev/github.com/kataras/iris/v14@v14.0.0). Для полного практического знакомства с рекомендуемым стеком v14 прочитайте [вводный учебник](TUTORIAL.md), а если вы переходите с v12, следуйте [Руководству по миграции](MIGRATION.md).

Предпочитаете учиться по коду? [**59 примеров**](_examples/README.md) проведут вас от первого маршрута до внедрения зависимостей, аутентификации, тестирования и точек расширения фреймворка. Каждый из них был скомпилирован и проверен по HTTP перед коммитом.

### Книга Iris

<a href="https://iris-go.com/book/"> <img alt="Book cover" src="https://iris-go.com/images/iris-book-cover-sm.jpg?v=14" /> </a>

Полная книга включает предисловие, 24 главы и эпилог, каждый пример кода сверен с исходным кодом фреймворка. Каждая глава открывается доступным введением для новичков и завершается подборкой материалов для дальнейшего чтения. Каждую главу можно бесплатно прочитать на [iris-go.com/docs](https://iris-go.com/docs/), а полное оглавление есть на [странице книги](https://iris-go.com/book/).

PDF тоже бесплатен. [Скачайте его напрямую](https://iris-go.com/iris-book.pdf): не нужно ни заполнять форму, ни что-либо запрашивать. Если книга окажется вам полезной, вы можете принять участие в развитии Iris, [став спонсором проекта](https://iris-go.com/donate/).

[![follow author on twitter](https://img.shields.io/twitter/follow/makismaropoulos?color=3D8AA3&logoColor=3D8AA3&style=for-the-badge&logo=twitter)](https://twitter.com/intent/follow?screen_name=makismaropoulos)

[![follow Iris web framework on twitter](https://img.shields.io/twitter/follow/iris_framework?color=ee7506&logoColor=ee7506&style=for-the-badge&logo=twitter)](https://twitter.com/intent/follow?screen_name=iris_framework)

[![follow Iris web framework on facebook](https://img.shields.io/badge/Follow%20%40Iris.framework-569-2D88FF.svg?style=for-the-badge&logo=facebook)](https://www.facebook.com/iris.framework)

## 🤖 AI-инструменты для Iris

Два AI-инструмента знают именно этот фреймворк, а не только то, что общая модель знает про Go.

[Живая AI-вики](https://codewiki.google/github.com/kataras/iris) бесплатна и работает в браузере: изучайте архитектуру или спрашивайте чат о коде.

Настольное приложение [Plexon AI](https://plexon.ai) для Windows, macOS и Linux создано автором Iris. В него встроен Iris skill ([iris-go.com/ai](https://iris-go.com/ai/)): API версии v14, цепочка builder, хелперы `rest`, отображение ошибок, SDK аутентификации и шаблоны тестов в двадцати одном справочном документе, которые ассистент подгружает по мере надобности, пока пишет ваш код. Включите skill на панели Skills или установите [персону Software Developer](https://plexon.ai/personas/software-developer/), которая включит его вместе с остальными инструментами разработки. [Скачать Plexon AI](https://plexon.ai/download/).

---

О спонсорах, участии в разработке, политике безопасности и лицензии читайте в [английском README](README.md).

Проект распространяется по [лицензии BSD 3-clause](LICENSE), так же, как и сам проект Go.
