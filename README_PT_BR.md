<!-- [![Black Lives Matter](https://iris-go.com/images/blacklivesmatter_banner.png)](https://support.eji.org/give/153413/#!/donation/checkout)-->

# Iris Web Framework <a href="README_GR.md"><img width="20px" src="https://iris-go.com/images/flag-greece.svg" /></a> <a href="README_FR.md"><img width="20px" src="https://iris-go.com/images/flag-france.svg" /></a> <a href="README_ZH.md"><img width="20px" src="https://iris-go.com/images/flag-china.svg" /></a> <a href="README_ES.md"><img width="20px" src="https://iris-go.com/images/flag-spain.png" /></a> <a href="README_FA.md"><img width="20px" src="https://iris-go.com/images/flag-iran.svg" /></a> <a href="README_RU.md"><img width="20px" src="https://iris-go.com/images/flag-russia.svg" /></a> <a href="README_KO.md"><img width="20px" src="https://iris-go.com/images/flag-south-korea.svg?v=12" /></a> <a href="README_PT_BR.md"><img width="20px" align="center" src="https://www.iris-go.com/images/flag-brazil.svg" /></a> <a href="README_JA.md"><img width="20px" height="20px" src="https://iris-go.com/images/flag-japan.svg" /></a>

[![build status](https://img.shields.io/github/actions/workflow/status/kataras/iris/ci.yml?branch=main&style=for-the-badge)](https://github.com/kataras/iris/actions/workflows/ci.yml) [![view examples](https://img.shields.io/badge/examples%20-270-a83adf.svg?style=for-the-badge&logo=go)](https://github.com/kataras/iris/tree/main/_examples) [![chat](https://img.shields.io/gitter/room/iris_go/community.svg?color=cc2b5e&logo=gitter&style=for-the-badge)](https://gitter.im/iris_go/community) <!--[![FOSSA Status](https://img.shields.io/badge/LICENSE%20SCAN-PASSING❤️-CD2956?style=for-the-badge&logo=fossa)](https://app.fossa.io/projects/git%2Bgithub.com%2Fkataras%2Firis?ref=badge_shield)--> [![donate](https://img.shields.io/badge/support-Iris-blue.svg?style=for-the-badge&logo=paypal)](https://iris-go.com/donate) <!--[![report card](https://img.shields.io/badge/report%20card-a%2B-ff3333.svg?style=for-the-badge)](https://goreportcard.com/report/github.com/kataras/iris)--><!--[![godocs](https://img.shields.io/badge/go-%20docs-488AC7.svg?style=for-the-badge)](https://pkg.go.dev/github.com/kataras/iris/v12@v12.2.11)--> <!-- [![release](https://img.shields.io/badge/release%20-v12.0-0077b3.svg?style=for-the-badge)](https://github.com/kataras/iris/releases) -->

<!-- <a href="https://iris-go.com"> <img align="right" src="https://iris-go.com/images/logo-w169.png"></a> -->

Iris é um framework web rápido, simples, mas completo e muito eficiente para Go.

Ele fornece uma base lindamente expressiva e fácil de usar para seu próximo site ou API.


```go
package main

import "github.com/kataras/iris/v12"

func main() {
  app := iris.New()
  app.Use(iris.Compression)

  app.Get("/", func(ctx iris.Context) {
    ctx.HTML("Hello <strong>%s</strong>!", "World")
  })

  app.Listen(":8080")
}
```

<!-- <details><summary>More with simple Handler</summary>

```go
package main

import "github.com/kataras/iris/v12"

type (
  request struct {
    Firstname string `json:"firstname"`
    Lastname  string `json:"lastname"`
  }

  response struct {
    ID      string `json:"id"`
    Message string `json:"message"`
  }
)

func main() {
  app := iris.New()
  app.Handle("PUT", "/users/{id:uuid}", updateUser)
  app.Listen(":8080")
}

func updateUser(ctx iris.Context) {
  id := ctx.Params().Get("id")

  var req request
  if err := ctx.ReadJSON(&req); err != nil {
    ctx.StopWithError(iris.StatusBadRequest, err)
    return
  }

  resp := response{
    ID:      id,
    Message: req.Firstname + " updated successfully",
  }
  ctx.JSON(resp)
}
```

> Read the [routing examples](https://github.com/kataras/iris/blob/main/_examples/routing) for more!

</details>

<details><summary>Handler with custom input and output arguments</summary>

[![https://github.com/kataras/iris/blob/main/_examples/dependency-injection/basic/main.go](https://user-images.githubusercontent.com/22900943/105253731-b8db6d00-5b88-11eb-90c1-0c92a5581c86.png)](https://twitter.com/iris_framework/status/1234783655408668672)

> Interesting? Read the [examples](https://github.com/kataras/iris/blob/main/_examples/dependency-injection).

</details>

<details><summary>Party Controller (NEW)</summary>

> Head over to the [full running example](https://github.com/kataras/iris/blob/main/_examples/routing/party-controller)!

</details>

<details><summary>MVC</summary>

```go
package main

import (
  "github.com/kataras/iris/v12"
  "github.com/kataras/iris/v12/mvc"
)

type (
  request struct {
    Firstname string `json:"firstname"`
    Lastname  string `json:"lastname"`
  }

  response struct {
    ID      uint64 `json:"id"`
    Message string `json:"message"`
  }
)

func main() {
  app := iris.New()
  mvc.Configure(app.Party("/users"), configureMVC)
  app.Listen(":8080")
}

func configureMVC(app *mvc.Application) {
  app.Handle(new(userController))
}

type userController struct {
  // [...dependencies]
}

func (c *userController) PutBy(id uint64, req request) response {
  return response{
    ID:      id,
    Message: req.Firstname + " updated successfully",
  }
}
```

Want to see more? Navigate through [mvc examples](_examples/mvc)!
</details>


<details><summary>API Guide <strong>HOT</strong></summary>

```go
package main

import (
  // [other packages...]

  "github.com/kataras/iris/v12"
)

func main() {
  iris.NewGuide().
    AllowOrigin("*").
    Compression(true).
    Health(true, "development", "kataras").
    Timeout(0, 20*time.Second, 20*time.Second).
    Middlewares(basicauth.New(...)).
    Services(
        // NewDatabase(),
        // NewPostgresRepositoryRegistry,
        // NewUserService,
    ).
    API("/users", new(UsersAPI)).
    Listen(":80")
}
```

</details>

<br/>

-->

Como um [Desenvolvedor Go](https://twitter.com/dkuye/status/1532087942696554497) disse uma vez, **Iris abrangeu tudo e se manteve forte ao longo dos anos**.

Alguns dos recursos que o Iris Web Framework oferece:

* HTTP/2 (Push, mesmo para dados incorporados)
* Middleware (Accesslog, Basicauth, CORS, gRPC, Anti-Bot hCaptcha, JWT, MethodOverride, ModRevision, Monitor, PPROF, Ratelimit, Anti-Bot reCaptcha, Recovery, RequestID, Rewrite)
* Versionamento de API
* Model-View-Controller
* Websockets
* gRPC
* Auto-HTTPS
* Suporte integrado para ngrok para colocar seu aplicativo na internet da maneira mais rápida
* Router único com caminho dinâmico como parametro com tipos padrões como :uuid, :string, :int... e a habilidade de criar o seu próprio router
* Compressão
* View Engines (HTML, Django, Handlebars, Pug/Jade e mais)
* Cria seu próprio Servidor de Arquivo e hospeda seu próprio servidor WebDAV
* Cache
* Localização (i18n, sitemap)
* Sessões
* Respostas Ricas (HTML, Text, Markdown, XML, YAML, Binary, JSON, JSONP, Protocol Buffers, MessagePack, Content Negotiation, Streaming, Server-Sent Events e mais)
* Compressão de resposta (gzip, deflate, brotli, snappy, s2)
* Requisições Ricas (Bind URL Query, Headers, Form, Text, XML, YAML, Binary, JSON, Validation, Protocol Buffers, MessagePack e mais)
* Injeção de dependência (MVC, Handlers, API Routers)
* Suite de testes
* E o mais importante... você obtém respostas rápidas e suporte desde o 1º dia até agora - são seis anos completos!

Aprenda com [o que os outros falam sobre Iris](https://www.iris-go.com/#review) e **[marque com uma estrela](https://github.com/kataras/iris/stargazers)** esse projeto de código aberto para apoiar o seu potencial.

[![](https://iris-go.com/images/reviews.gif)](https://iris-go.com/testimonials/)

[![Benchmarks: Jul 18, 2020 at 10:46am (UTC)](https://iris-go.com/images/benchmarks.svg)](https://github.com/kataras/server-benchmarks)

## 👑 <a href="https://iris-go.com/donate">Apoiadores</a>

Com a sua ajuda, nós podemos melhorar o desenvolvimento web de Código Aberto para todos !

> [@github](https://github.com/github) agora está patrocinando você por $550,00 uma vez.
>
> Uma nota do seu novo patrocinador: 
>
> Para comemorar o Mês do Mantenedor, queremos agradecer por tudo que você faz pela comunidade de código aberto. Confira nossa postagem no blog para saber mais sobre como o GitHub está investindo em mantenedores

> Doações direto da [China](https://github.com/kataras/iris/issues/1870#issuecomment-1101418349) agora são aceitas!

## 📖 Aprenda sobre o Iris Web Framework

### Instalação

O único requisito é a [Linguagem de programação Go](https://go.dev/dl/).

#### Criar um novo projeto

```sh
$ mkdir myapp
$ cd myapp
$ go mod init myapp
$ go get github.com/kataras/iris/v12@latest # or @v12.2.11
```

<details><summary>Instalar num projeto existente</summary>

```sh
$ cd myapp
$ go get github.com/kataras/iris/v12@latest
```

**Run**

```sh
$ go mod tidy -compat=1.20 # -compat="1.20" for windows.
$ go run .
```

</details>

![](https://www.iris-go.com/images/gifs/install-create-iris.gif)

Iris contém extensa e completa **[documentação](https://www.iris-go.com/docs)**, o que torna fácil o começo com o framework.

<!-- Iris contains extensive and thorough **[wiki](https://github.com/kataras/iris/wiki)** making it easy to get started with the framework. -->

<!-- ![](https://media.giphy.com/media/Ur8iqy9FQfmPuyQpgy/giphy.gif) -->

Para obter uma documentação técnica mais detalhada, você pode acessar nosso [godocs](https://pkg.go.dev/github.com/kataras/iris/v12@main). E para executar o código você sempre pode visitar os subdiretórios do diretório [./_examples](_examples).

### Você gosta de ler enquanto viaja ?

<a href="https://iris-go.com/#book"> <img alt="Book cover" src="https://iris-go.com/images/iris-book-cover-sm.jpg?v=12" /> </a>

[![follow author on twitter](https://img.shields.io/twitter/follow/makismaropoulos?color=3D8AA3&logoColor=3D8AA3&style=for-the-badge&logo=twitter)](https://twitter.com/intent/follow?screen_name=makismaropoulos)

[![follow Iris web framework on twitter](https://img.shields.io/twitter/follow/iris_framework?color=ee7506&logoColor=ee7506&style=for-the-badge&logo=twitter)](https://twitter.com/intent/follow?screen_name=iris_framework)

[![follow Iris web framework on facebook](https://img.shields.io/badge/Follow%20%40Iris.framework-569-2D88FF.svg?style=for-the-badge&logo=facebook)](https://www.facebook.com/iris.framework)

Você pode [solicitar](https://www.iris-go.com/#ebookDonateForm) o acesso ao **Iris E-Book** de forma online e também no formato PDF (Nova edição, **future v12.2.0+**) hoje today e se antecipar no desenvolvimento do Iris.

## 🙌 Contribuidores

Adoraríamos ver sua contribuição para o Iris Web Framework! Para mais informações sobre como contribuir com o projeto Iris, consulte o arquivo [CONTRIBUTING.md](CONTRIBUTING.md).

[Lista de todos os Contribuidores](https://github.com/kataras/iris/graphs/contributors)

## 🛡Vulnerabilidades de segurança

Se você descobrir alguma vulnerabilidade de segurança dentro do Iris, por favor, envie um email para [iris-go@outlook.com](mailto:iris-go@outlook.com). Todas as vulnerabilidades de segurança serão prontamente tratadas.

## 📝 Licença
Este projeto está licenciado sob a [Licença BSD 3-clause](LICENSE), assim como o próprio projeto Go.

O nome do projeto "Iris" foi inspirado pela mitologia Grega.
<!-- ## Stargazers over time

[![Stargazers over time](https://starchart.cc/kataras/iris.svg)](https://starchart.cc/kataras/iris) -->
