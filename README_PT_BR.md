# Iris Web Framework <a href="README.md"> <img width="20px" src="https://iris-go.com/images/flag-unitedkingdom.svg?v=10" /></a> <a href="README_ZH.md"> <img width="20px" src="https://iris-go.com/images/flag-china.svg?v=10" /></a> <a href="README_RU.md"><img width="20px" src="https://iris-go.com/images/flag-russia.svg?v=10" /></a> <a href="README_ID.md"> <img width="20px" src="https://iris-go.com/images/flag-indonesia.svg?v=10" /></a> <a href="README_GR.md"><img width="20px" src="https://iris-go.com/images/flag-greece.svg?v=10" /></a> <a href="README_JPN.md"><img width="20px" src="https://iris-go.com/images/flag-japan.svg?v=10" /></a>

<a href="https://iris-go.com"> <img align="right" width="169px" src="https://iris-go.com/images/icon.svg?v=a" title="logo created by @merry.dii" /> </a>

[![build status](https://img.shields.io/travis/kataras/iris/master.svg?style=flat-square)](https://travis-ci.org/kataras/iris)<!-- [![release](https://img.shields.io/github/release/kataras/iris.svg?style=flat-square)](https://github.com/kataras/iris/releases)--> [![report card](https://img.shields.io/badge/report%20card-a%2B-ff3333.svg?style=flat-square)](http://goreportcard.com/report/kataras/iris) [![vscode-iris](https://img.shields.io/badge/ext%20-vscode-0c77e3.svg?style=flat-square)](https://marketplace.visualstudio.com/items?itemName=kataras2006.iris)<!--[![github closed issues](https://img.shields.io/github/issues-closed-raw/kataras/iris.svg?style=flat-square)](https://github.com/kataras/iris/issues?q=is%3Aissue+is%3Aclosed)--> [![chat](https://img.shields.io/badge/community-%20chat-00BCD4.svg?style=flat-square)](https://kataras.rocket.chat/channel/iris) [![view examples](https://img.shields.io/badge/learn%20by-examples-0077b3.svg?style=flat-square)](https://github.com/kataras/iris/tree/master/_examples/routing) [![release](https://img.shields.io/badge/release%20-v11.1-0077b3.svg?style=flat-square)](https://github.com/kataras/iris/releases)

Iris é um framework rápido, simples porém completo e muito eficiente para a linguagem Go.

Além disso, Iris proporciona uma base sólida que capricha na expressividade e facilidade de uso para seu próximo site ou API.

Por último, Iris é um framework equivalente ao expressjs no ecossistema da linguagem de programação Go.

Veja o que [as pessoas estão dizendo sobre o Iris](#support) e [deixe uma estrela](https://github.com/kataras/iris/stargazers) nesse repositório do github para se [manter atualizado](https://facebook.com/iris.framework).

## Apoiadores

Muito obrigado a todos que nos apoiam! 🙏 [Apoie a gente!](https://iris-go.com/donate)


<a href="https://iris-go.com/donate" target="_blank"><img src="https://iris-go.com/backers.svg?v=2"/></a>

```sh
$ cat example.go
```

```go
package main

import "github.com/kataras/iris"

func main() {
    app := iris.New()
    // Carrega todos os templates da pasta "./views"
    // cuja extensão é ".html" e parseie-os utilizando
    // a biblioteca `html/template`.
    app.RegisterView(iris.HTML("./views", ".html"))

    // Method:    GET
    // Resource:  http://localhost:8080
    app.Get("/", func(ctx iris.Context) {
        // Associa {{.message}} a "Hello world!"
        ctx.ViewData("message", "Hello world!")
        // Renderiza o template: ./views/hello.html
        ctx.View("hello.html")
    })

    // Method:    GET
    // Resource:  http://localhost:8080/user/42
    //
    // Deseja utilizar uma expressão regular ?
    // É fácil,
    // é só marcar o type to parametro como 'string'
    // e utilizar sua macro `regexp`, i.e:
    // app.Get("/user/{id:string regexp(^[0-9]+$)}")
    app.Get("/user/{id:long}", func(ctx iris.Context) {
        userID, _ := ctx.Params().GetInt64("id")
        ctx.Writef("User ID: %d", userID)
    })

    // Inicializa o servidor utilizando um endereço de rede.
    app.Run(iris.Addr(":8080"))
}
```

> Aprenda mais sobre tipos dos parametros da URI clicando [aqui](_examples/routing/dynamic-path/main.go#L31)

```html
<!-- arquivo: ./views/hello.html -->
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

## Instalação

O único pré requisito é a [Linguagem de Programação GO](https://golang.org/dl/)

```sh
$ go get -u github.com/kataras/iris
```

Iris lança mão da [pasta vendor](https://docs.google.com/document/d/1Bz5-UB7g2uPBdOx-rw5t9MxJwkfpx90cqG9AFL0JAYo). Dessa forma você conseguirá obter builds reprodutíveis já que esse método impede que nomes no branch upstream sejam renomeados ou deletados.

[![Iris vs .NET Core(C#) vs Node.js (Express)](https://iris-go.com/images/benchmark-new-gray.png)](_benchmarks/README_UNIX.md)

_Atualizado em : [Terça, 21 de Novembro de 2017](_benchmarks/README_UNIX.md)_

<details>
<summary>Benchmarks de fonte third-party acerca dos frameworks web</summary>

![Comparison with other frameworks](https://raw.githubusercontent.com/smallnest/go-web-framework-benchmark/4db507a22c964c9bc9774c5b31afdc199a0fe8b7/benchmark.png)

</details>

## Apoie

- [HISTORY](HISTORY.md#su-18-november-2018--v1110) o arquivo HISTORY é o seu melhor amigo, ele contém informações sobre as últimas features e mudanças.
- Econtrou algum bug ? Poste-o nas [issues](https://github.com/kataras/iris/issues)
- Possui alguma dúvida ou gostaria de falar com alguém experiente para resolver seu problema em tempo real ? Junte-se ao [chat da nossa comunidade](https://chat.iris-go.com).
- Complete nosso formulário de experiência do usuário clicando [aqui](https://docs.google.com/forms/d/e/1FAIpQLSdCxZXPANg_xHWil4kVAdhmh7EBBHQZ_4_xSZVDL-oCC_z5pA/viewform?usp=sf_link)
- Gostou do framework ? Deixe um Tweet sobre ele! Veja o que os outros já disseram:

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

Para mais informações sobre como contribuir para o projeto Iris leia por favor o arquivo [CONTRIBUTING.md](CONTRIBUTING.md).

[Lista de todos os Contribuintes](https://github.com/kataras/iris/graphs/contributors)

## Aprenda

Primeiramente, a melhor maneira de começar a aprender um framework é
aprender os fundamentos da linguagem de programação em questão e
as funções principais da biblioteca `http`, se seu app é um projeto
pessoal muito simples que exija performance e manutenção contínua
é provável que você consiga seguir adiante apenas com a biblioteca
padrão. Feito isso, você pode seguir as seguintes diretrizes:

- Navegue por **100+1** **[exemplos](_examples)** e os [iris starter kits](#iris-starter-kits) que criamos para você
- Leia os [godocs](https://godoc.org/github.com/kataras/iris) para mais detalhes
- Prepare um chá ou cafezinho, ou o que lhe for mais conveniente, e leia alguns [artigos](#articles) que achamos para você

### Iris starter kits

<!-- table form 
| Description | Link |
| -----------|-------------|
| Hasura hub starter project with a ready to deploy golang helloworld webapp with IRIS! | https://hasura.io/hub/project/hasura/hello-golang-iris |
| Web app básico utilizando o Iris |https://github.com/gauravtiwari/go_iris_app |
| Uma mini rede social criada com o incrível Iris💖💖 | https://github.com/iris-contrib/Iris-Mini-Social-Network |
| Iris isomorphic react/hot reloadable/redux/css-modules starter kit | https://github.com/iris-contrib/iris-starter-kit |
| Projeto demo usando react com typescript e Iris | https://github.com/ionutvilie/react-ts |
| Plataforma de Gerenciamento de Localização auto hospedada criada com Iris e Angular | https://github.com/iris-contrib/parrot |
| Iris + Docker e Kubernetes | https://github.com/iris-contrib/cloud-native-go |
| Quickstart do Iris com Nanobox | https://guides.nanobox.io/golang/iris/from-scratch |
-->

1. [A basic CRUD API in golang with Iris](https://github.com/jebzmos4/Iris-golang)
2. [Web app básico utilizando o Iris](https://github.com/gauravtiwari/go_iris_app)
3. [Uma mini rede social criada com o incrível Iris💖💖] (https://github.com/iris-contrib/Iris-Mini-Social-Network)
4. [Iris isomorphic react/hot reloadable/redux/css-modules starter kit](https://github.com/iris-contrib/iris-starter-kit)
5. [Projeto demo usando react com typescript e Iris](https://github.com/ionutvilie/react-ts)
6. [Plataforma de Gerenciamento de Localização auto hospedada criada com Iris e Angular](https://github.com/iris-contrib/parrot)
7. [Iris + Docker e Kubernetes](https://github.com/iris-contrib/cloud-native-go)
8. [Quickstart do Iris com Nanobox](https://guides.nanobox.io/golang/iris/from-scratch)
9. [Um projeto Hasura para iniciantes pronto para o deply com um app Golang hello-world utilizando o IRIS](https://hasura.io/hub/project/hasura/hello-golang-iris)

> Voce criou algo parecido ? [Informe-nos](https://github.com/kataras/iris/pulls)!

### Middleware

Iris tem uma ótima coleção de handlers[[1]](middleware/)[[2]](https://github.com/iris-contrib/middleware) os quais você pode utilizar lado a lado com seus web apps. Entretanto, você não esta limitado a eles - você pode utilizar qualquer middleware de terceiros desde que seja compatível com a biblioteca [net/http](https://golang.org/pkg/net/http/), [_examples/convert-handlers](_examples/convert-handlers) é um exemplo que pode ser tomado como base para tal.

Iris, ao contrário dos demais, é 100% compatível com os padrões e esse é o motivo pelo qual a maioria das grandes empresas que inserem Go em seu fluxo operacional, tal qual a famosa US Television Network, usam e confiam no Iris; Ele é atualizado com frequencia e sempre estará alinhado com o padrão da biblioteca `net/http` que é periodicamente modernizada pelos autores da linguagem Go a cada novo release.

### Artigos

* [CRUD REST API in Iris (a framework for golang)](https://medium.com/@jebzmos4/crud-rest-api-in-iris-a-framework-for-golang-a5d33652401e)
* [Um aplicação Todo utilizando Iris e Vue.js](https://hackernoon.com/a-todo-mvc-application-using-iris-and-vue-js-5019ff870064)
* [Um projeto Hasura para iniciantes pronto para o deply com um app Golang hello-world utilizando o IRIS](https://bit.ly/2lmKaAZ)
* [Top 6 frameworks web do Go em 2017](https://blog.usejournal.com/top-6-web-frameworks-for-go-as-of-2017-23270e059c4b)
* [Framework Iris + MongoDB](https://medium.com/go-language/iris-go-framework-mongodb-552e349eab9c)
* [Como criar um formulário de upload de arquivos com DropzoneJS e Go](https://hackernoon.com/how-to-build-a-file-upload-form-using-dropzonejs-and-go-8fb9f258a991)
* [Como mostrar arquivos existentes no servidor utilizando DropzoneJS e Go](https://hackernoon.com/how-to-display-existing-files-on-server-using-dropzonejs-and-go-53e24b57ba19)
* [Iris,um web framework modular](https://medium.com/@corebreaker/iris-web-cd684b4685c7)
* [Go vs .NET Core em termos de performance HTTP](https://medium.com/@kataras/go-vs-net-core-in-terms-of-http-performance-7535a61b67b8)
* [Iris Go vs .NET Core Kestrel em termos de performance HTTP](https://hackernoon.com/iris-go-vs-net-core-kestrel-in-terms-of-http-performance-806195dc93d5)
* [Como transformar um aparelho Android em um servidor web](https://twitter.com/ThePracticalDev/status/892022594031017988)
* [Fazendo Deploy de um app Iris na hasura](https://medium.com/@HasuraHQ/deploy-an-iris-golang-app-with-backend-apis-in-minutes-25a559bf530b)
* [Um serviço encurtador de URL utilizando Go, Iris e Bolt](https://medium.com/@kataras/a-url-shortener-service-using-go-iris-and-bolt-4182f0b00ae7)

### Video Aulas

* [Daily Coding - Web Framework Golang: Iris Framework]( https://www.youtube.com/watch?v=BmOLFQ29J3s) por WarnabiruTV, fonte: youtube, custo: **GRATUITO**
* [Tutorial Golang MVC dengan Iris Framework & Mongo DB](https://www.youtube.com/watch?v=uXiNYhJqh2I&list=PLMrwI6jIZn-1tzskocnh1pptKhVmWdcbS) (19 ate o momento) por Musobar Media, fonte: youtube, custo: **GRATUITO**
* [Go/Golang 27 - Iris framework : Routage de base](https://www.youtube.com/watch?v=rQxRoN6ub78) por stephgdesign, fonte: youtube, custo: **GRATUITO**
* [Go/Golang 28 - Iris framework : Templating](https://www.youtube.com/watch?v=nOKYV073S2Y) por stephgdesignn, fonte: youtube, custo: **GRATUITO**
* [Go/Golang 29 - Iris framework : Paramètres](https://www.youtube.com/watch?v=K2FsprfXs1E) por stephgdesign, fonte: youtube, custo: **GRATUITO**
* [Go/Golang 30 - Iris framework : Les middelwares](https://www.youtube.com/watch?v=BLPy1So6bhE) por stephgdesign, fonte: youtube, custo: **GRATUITO**
* [Go/Golang 31 - Iris framework : Les sessions](https://www.youtube.com/watch?v=RnBwUrwgEZ8) por stephgdesign, fonte: youtube, custo: **GRATUITO**

### Seja contratado

Várias empresas e start-ups estão procurando por desenvolvedores web que sabem Go e possuam experiência com Iris como pré requisito, todos os dias estamos procurando informações sobre empregos e postando na nossa [página do facebook](https://www.facebook.com/iris.framework), de um like na página para ser notificado.

## Licença

Iris é licenciado sob a [Licença 3-Clause BSD](LICENSE). Iris é um software 100% gratuito e open-source.

Caso haja quaisquer dúvidas em relação a licença favor enviar um [e-mail](mailto:kataras2006@hotmail.com?subject=Iris%20License).
