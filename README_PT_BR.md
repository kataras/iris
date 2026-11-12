# <a href="https://iris-go.com"><img src="https://iris-go.com/images/logo-new-lq-45.png"></a> Iris Web Framework <a href="README_GR.md"><img width="20px" src="https://iris-go.com/images/flag-greece.svg" /> <a href="README_JA.md"><img width="20px" height="20px" src="https://iris-go.com/images/flag-japan.svg" /></a> </a> <a href="README_FR.md"><img width="20px" src="https://iris-go.com/images/flag-france.svg" /></a> <a href="README_ZH_HANT.md"><img width="20px" src="https://iris-go.com/images/flag-taiwan.svg" /></a> <a href="README_ZH_HANS.md"><img width="20px" src="https://iris-go.com/images/flag-china.svg" /></a> <a href="README_ES.md"><img width="20px" src="https://iris-go.com/images/flag-spain.png" /></a> <a href="README_FA.md"><img width="20px" src="https://iris-go.com/images/flag-iran.svg" /></a> <a href="README_RU.md"><img width="20px" src="https://iris-go.com/images/flag-russia.svg" /></a> <a href="README_KO.md"><img width="20px" src="https://iris-go.com/images/flag-south-korea.svg?v=12" /></a> <a href="README_PT_BR.md"><img width="20px" height="20px" src="https://iris-go.com/images/flag-brazil.svg" /></a> <a href="README_VN.md"><img width="20px" height="20px" src="https://iris-go.com/images/flag-vietnam.svg" /></a>

[![build status](https://img.shields.io/github/actions/workflow/status/kataras/iris/ci.yml?branch=main&style=for-the-badge)](https://github.com/kataras/iris/actions/workflows/ci.yml) [![release](https://img.shields.io/badge/release%20-v14.0-0077b3.svg?style=for-the-badge)](https://github.com/kataras/iris/releases) [![view examples](https://img.shields.io/badge/examples%20-59-a83adf.svg?style=for-the-badge&logo=go)](_examples/README.md) [![godocs](https://img.shields.io/badge/go-%20docs-488AC7.svg?style=for-the-badge)](https://pkg.go.dev/github.com/kataras/iris/v14@v14.0.0) [![chat](https://img.shields.io/gitter/room/iris_go/community.svg?color=cc2b5e&logo=gitter&style=for-the-badge)](https://gitter.im/iris_go/community) [![donate](https://img.shields.io/badge/support-Iris-blue.svg?style=for-the-badge&logo=paypal)](https://iris-go.com/donate)

Iris é um framework web para Go, rápido e com tudo o que você precisa incluído. Ele dá à sua próxima API ou site um roteamento expressivo com parâmetros de rota tipados, um único lugar para o tratamento de erros, injeção de dependências que mantém o código testável e padrões de produção seguros desde o início.

A **versão 14** é o maior lançamento da história do projeto. O framework foi reconstruído em torno dos generics de Go: um builder de aplicações verificado em compilação, helpers de requisição tipados que validam por você, controllers resolvidos pelo contêiner de dependências e um pipeline de erros que você configura uma única vez. O [HISTORY.md](HISTORY.md) registra todas as novidades; o [Guia de Migração](MIGRATION.md) atualiza aplicações v12 em uma única sessão.

> 🤖 **Pergunte à IA sobre o Iris.** O [Code Wiki do Iris](https://codewiki.google/github.com/kataras/iris), gratuito e criado pelo Google, é gerado a partir deste repositório e permanece sincronizado com ele, commit a commit. Navegue pela arquitetura com diagramas, ou faça ao chat perguntas em linguagem natural sobre o código e receba respostas fundamentadas no código-fonte real.

Veja o que [outros dizem sobre o Iris](https://www.iris-go.com/#review) e dê uma **[estrela](https://github.com/kataras/iris/stargazers)** a este projeto de código aberto para apoiar o seu desenvolvimento. Curioso sobre a velocidade? Confira os [benchmarks de servidores](https://github.com/kataras/server-benchmarks) independentes.

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

<details><summary>A stack de produção, em uma única expressão</summary>

Aplicações reais usam o `iris.NewBuilder`: uma cadeia verificada em compilação que conecta CORS, compressão, logs de acesso, um endpoint de health, o seu mapa de erros, os seus serviços e os seus grupos de API. O compilador rejeita a ordem errada dos passos.

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

Os handlers ficam em seis linhas, porque a decodificação, a validação e a renderização de erros acontecem em um único lugar configurado:

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

O [Tutorial de Introdução](TUTORIAL.md) constrói um serviço completo e testado sobre essa stack em um único documento.

</details>

Como disse certa vez um [desenvolvedor Go](https://twitter.com/dkuye/status/1532087942696554497), **o Iris cobre você em todas as frentes e se mantém firme ao longo dos anos**.

O que você ganha com o Iris v14:

* Roteamento com parâmetros de rota tipados (`{id:uuid}`, `{n:int}`, `{addr:email}` e mais), macros personalizadas e restrições. Entradas inválidas são rejeitadas antes de o seu handler executar.
* `iris.NewBuilder`: CORS, compressão, log de acessos, endpoint de health, mapeamento de erros, serviços e grupos de API montados em uma cadeia verificada em compilação.
* Injeção de dependências: os construtores declaram o que precisam, o contêiner fornece. Hooks de inicialização `Initializer`, closers de desligamento automáticos.
* Os helpers `rest` e um mapa central de erros: handlers enxutos, erros canônicos legíveis por máquina, detalhes internos nunca vazam para os clientes.
* Payloads de requisição que se validam sozinhos: implemente `Validate() error` e o `ctx.BindJSON` o executa após a decodificação.
* Trinta e um middlewares embutidos: accesslog, API keys, basicauth, bodylimit, cache, CORS, hCaptcha e reCAPTCHA, JWT, monitor, pprof, rate limiting, recovery, request IDs, rewrite, sessions, versioning e mais.
* Autenticação em qualquer escala: Basic Auth em uma linha, chaves de API, um SDK de JWT genérico, um servidor de identidade completo e um provedor OAuth2.
* Controllers MVC construídos sobre os generics de Go.
* WebSockets, serviço gRPC e versionamento de API.
* Engines de visualização: HTML, Django, Handlebars, Pug e mais via iris-contrib/views.
* Localização com pluralização e detecção de idioma, além de sitemaps.
* Sessões com padrões seguros de cookies (`Secure`, `SameSite=Lax` de fábrica).
* Respostas ricas: JSON, JSONP, XML, YAML, Markdown, Protocol Buffers, MessagePack, negociação de conteúdo, streaming e Server-Sent Events.
* Backends de IA: o pacote `ai` renderiza e retransmite por HTTP a saída em streaming dos modelos (deltas de texto, rastros de raciocínio, fragmentos de chamadas de ferramentas, uso). Não depende de nenhum provedor e não traz o SDK de nenhum deles.
* Um cliente HTTP, o pacote `client`, para chamar outros serviços com as mesmas regras de JSON que o lado do servidor usa.
* Compressão de respostas: gzip, deflate, brotli, snappy, s2, zstd.
* TLS com Let's Encrypt, e túnel ngrok embutido (opt-in: `import _ "github.com/kataras/iris/v14/core/server/ngrok"`) para colocar uma aplicação de desenvolvimento na internet em segundos.
* Testes HTTP em memória: testes da aplicação completa sem portas, sem rede e sem sleeps.
* Um livro completo que você pode ler online e baixar de graça, um tutorial prático e um wiki de IA ao vivo (veja Aprendendo Iris abaixo).
* E o mais importante: você recebe respostas rápidas e suporte, do primeiro dia até hoje.

## 📖 Aprendendo Iris

> Novo no Iris? Comece pelo [Tutorial de Introdução](TUTORIAL.md): um único documento que leva você do `go mod init` a uma API REST testada e com formato de produção. Atualizando da v12? Veja o [Guia de Migração](MIGRATION.md). Dúvidas em qualquer ponto? Pergunte ao **wiki de IA ao vivo**, gratuito, em [codewiki.google/github.com/kataras/iris](https://codewiki.google/github.com/kataras/iris), gerado pelo Google a partir do código-fonte do Iris e mantido em sincronia com ele, com um chat embutido que responde perguntas sobre o código.

### Instalação

O único requisito é a [linguagem de programação Go](https://go.dev/dl/), versão 1.27 ou superior.

#### Crie um novo projeto

```sh
$ mkdir myapp
$ cd myapp
$ go mod init myapp
$ go get github.com/kataras/iris/v14@latest # or @v14.0.0
```

<details><summary>Instalar em um projeto existente</summary>

```sh
$ cd myapp
$ go get github.com/kataras/iris/v14@latest
```

**Executar**

```sh
$ go mod tidy
$ go run .
```

</details>

![](https://www.iris-go.com/images/gifs/install-create-iris.gif)

O Iris traz uma **[documentação](https://www.iris-go.com/docs)** extensa e detalhada que facilita os primeiros passos com o framework.

Para uma documentação técnica mais detalhada, visite os nossos [godocs](https://pkg.go.dev/github.com/kataras/iris/v14@v14.0.0). Para uma introdução completa e prática à stack recomendada da v14, leia o [Tutorial de Introdução](TUTORIAL.md), e se você está atualizando da v12, siga o [Guia de Migração](MIGRATION.md).

Prefere aprender pelo código? Os [**59 exemplos**](_examples/README.md) vão de uma primeira rota até injeção de dependências, autenticação, testes e os pontos de extensão do framework. Cada um deles foi compilado e exercitado via HTTP antes de ser commitado.

### O Livro do Iris

<a href="https://iris-go.com/book/"> <img alt="Book cover" src="https://iris-go.com/images/iris-book-cover-sm.jpg?v=14" /> </a>

O livro completo tem um prefácio, 24 capítulos e um epílogo, com cada exemplo de código verificado contra o código-fonte do framework. Cada capítulo abre com um contexto em linguagem simples para iniciantes e fecha com leituras adicionais selecionadas. Todos os capítulos podem ser lidos de graça em [iris-go.com/docs](https://iris-go.com/docs/); a [página do livro](https://iris-go.com/book/) traz o índice completo.

O PDF também é gratuito. [Baixe diretamente](https://iris-go.com/iris-book.pdf): não há formulário para preencher nem nada para solicitar. Se o livro for útil para você, [patrocinar o projeto](https://iris-go.com/donate/) é a forma de participar do desenvolvimento do Iris.

[![follow author on twitter](https://img.shields.io/twitter/follow/makismaropoulos?color=3D8AA3&logoColor=3D8AA3&style=for-the-badge&logo=twitter)](https://twitter.com/intent/follow?screen_name=makismaropoulos)

[![follow Iris web framework on twitter](https://img.shields.io/twitter/follow/iris_framework?color=ee7506&logoColor=ee7506&style=for-the-badge&logo=twitter)](https://twitter.com/intent/follow?screen_name=iris_framework)

[![follow Iris web framework on facebook](https://img.shields.io/badge/Follow%20%40Iris.framework-569-2D88FF.svg?style=for-the-badge&logo=facebook)](https://www.facebook.com/iris.framework)

## 🤖 Ferramentas de IA para o Iris

Duas ferramentas de IA conhecem este framework especificamente, além do que um modelo genérico sabe sobre Go.

A [wiki de IA ao vivo](https://codewiki.google/github.com/kataras/iris) é gratuita e roda no navegador: explore a arquitetura ou pergunte ao chat sobre o código.

O [Plexon AI](https://plexon.ai) é um assistente de desktop para Windows, macOS e Linux, criado pelo autor do Iris. Ele já vem com a skill de Iris ([iris-go.com/ai](https://iris-go.com/ai/)): a API da v14, a cadeia do builder, os helpers `rest`, o mapeamento de erros, o SDK de autenticação e os padrões de teste, em vinte e um documentos de referência que o assistente carrega conforme precisa enquanto escreve seu código. Ative a skill pelo painel Skills ou instale a [persona Software Developer](https://plexon.ai/personas/software-developer/), que a ativa junto com o restante das ferramentas. [Baixe o Plexon AI](https://plexon.ai/download/).

Para apoiadores, contribuição, política de segurança e licença, veja o [README em inglês](README.md).

Este projeto está licenciado sob a [licença BSD 3-clause](LICENSE), assim como o próprio projeto Go.
