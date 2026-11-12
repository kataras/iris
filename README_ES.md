# <a href="https://iris-go.com"><img src="https://iris-go.com/images/logo-new-lq-45.png"></a> Iris Web Framework <a href="README_GR.md"><img width="20px" src="https://iris-go.com/images/flag-greece.svg" /> <a href="README_JA.md"><img width="20px" height="20px" src="https://iris-go.com/images/flag-japan.svg" /></a> </a> <a href="README_FR.md"><img width="20px" src="https://iris-go.com/images/flag-france.svg" /></a> <a href="README_ZH_HANT.md"><img width="20px" src="https://iris-go.com/images/flag-taiwan.svg" /></a> <a href="README_ZH_HANS.md"><img width="20px" src="https://iris-go.com/images/flag-china.svg" /></a> <a href="README_ES.md"><img width="20px" src="https://iris-go.com/images/flag-spain.png" /></a> <a href="README_FA.md"><img width="20px" src="https://iris-go.com/images/flag-iran.svg" /></a> <a href="README_RU.md"><img width="20px" src="https://iris-go.com/images/flag-russia.svg" /></a> <a href="README_KO.md"><img width="20px" src="https://iris-go.com/images/flag-south-korea.svg?v=12" /></a> <a href="README_PT_BR.md"><img width="20px" height="20px" src="https://iris-go.com/images/flag-brazil.svg" /></a> <a href="README_VN.md"><img width="20px" height="20px" src="https://iris-go.com/images/flag-vietnam.svg" /></a>

[![build status](https://img.shields.io/github/actions/workflow/status/kataras/iris/ci.yml?branch=main&style=for-the-badge)](https://github.com/kataras/iris/actions/workflows/ci.yml) [![release](https://img.shields.io/badge/release%20-v14.0-0077b3.svg?style=for-the-badge)](https://github.com/kataras/iris/releases) [![view examples](https://img.shields.io/badge/examples%20-59-a83adf.svg?style=for-the-badge&logo=go)](_examples/README.md) [![godocs](https://img.shields.io/badge/go-%20docs-488AC7.svg?style=for-the-badge)](https://pkg.go.dev/github.com/kataras/iris/v14@v14.0.0) [![chat](https://img.shields.io/gitter/room/iris_go/community.svg?color=cc2b5e&logo=gitter&style=for-the-badge)](https://gitter.im/iris_go/community) [![donate](https://img.shields.io/badge/support-Iris-blue.svg?style=for-the-badge&logo=paypal)](https://iris-go.com/donate)

Iris es un framework web para Go, rápido y con todo lo necesario incluido. Le da a tu próxima API o sitio web un enrutamiento expresivo con parámetros de ruta tipados, un único lugar para el manejo de errores, inyección de dependencias que mantiene el código testeable y valores de producción seguros por defecto.

La **versión 14** es el mayor lanzamiento en la historia del proyecto. El framework fue reconstruido en torno a los genéricos de Go: un builder de aplicaciones verificado en compilación, helpers de peticiones tipados que validan por ti, controladores resueltos por el contenedor de dependencias y una tubería de errores que configuras una sola vez. [HISTORY.md](HISTORY.md) registra todas las novedades; la [Guía de Migración](MIGRATION.md) actualiza aplicaciones v12 en una sola sesión.

> 🤖 **Pregúntale a la IA sobre Iris.** El [Code Wiki de Iris](https://codewiki.google/github.com/kataras/iris), gratuito y de Google, se genera a partir de este repositorio y se mantiene sincronizado con él, commit a commit. Explora la arquitectura con diagramas, o hazle a su chat preguntas en lenguaje natural sobre el código y recibe respuestas fundamentadas en el código fuente real.

Descubre lo que [otros dicen sobre Iris](https://www.iris-go.com/#review) y dale una **[estrella](https://github.com/kataras/iris/stargazers)** a este proyecto de código abierto para apoyar su desarrollo. ¿Te interesa la velocidad? Consulta los [benchmarks de servidores](https://github.com/kataras/server-benchmarks) independientes.

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

<details><summary>La pila de producción, en una sola expresión</summary>

Las aplicaciones reales usan `iris.NewBuilder`: una cadena verificada en compilación que conecta CORS, compresión, logs de acceso, un endpoint de salud, tu mapa de errores, tus servicios y tus grupos de API. El compilador rechaza un orden de pasos incorrecto.

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

Los handlers se quedan en seis líneas, porque la decodificación, la validación y el renderizado de errores ocurren en un único lugar configurado:

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

El [Tutorial de Inicio](TUTORIAL.md) construye un servicio completo y probado sobre esta pila en un solo documento.

</details>

Como dijo una vez un [desarrollador de Go](https://twitter.com/dkuye/status/1532087942696554497), **Iris te cubre en todos los frentes y se mantiene firme con el paso de los años**.

Lo que obtienes con Iris v14:

* Enrutamiento con parámetros de ruta tipados (`{id:uuid}`, `{n:int}`, `{addr:email}` y más), macros personalizadas y restricciones. La entrada inválida se rechaza antes de que se ejecute tu handler.
* `iris.NewBuilder`: CORS, compresión, registro de accesos, endpoint de salud, mapeo de errores, servicios y grupos de API ensamblados en una cadena verificada en compilación.
* Inyección de dependencias: los constructores declaran lo que necesitan y el contenedor lo provee. Hooks de arranque `Initializer`, closers de apagado automáticos.
* Los helpers `rest` y un mapa central de errores: handlers ligeros, errores canónicos legibles por máquinas, los detalles internos nunca se filtran a los clientes.
* Payloads de petición que se validan solos: implementa `Validate() error` y `ctx.BindJSON` lo ejecuta tras la decodificación.
* Treinta y un middleware integrados: accesslog, API keys, basicauth, bodylimit, cache, CORS, hCaptcha y reCAPTCHA, JWT, monitor, pprof, rate limiting, recovery, request IDs, rewrite, sessions, versioning y más.
* Autenticación a cualquier escala: Basic Auth en una línea, claves de API, un SDK de JWT genérico, un servidor de identidad completo y un proveedor OAuth2.
* Controladores MVC construidos sobre los genéricos de Go.
* WebSockets, servicio gRPC y versionado de API.
* Motores de vistas: HTML, Django, Handlebars, Pug y más a través de iris-contrib/views.
* Localización con pluralización y detección de idioma, además de sitemaps.
* Sesiones con valores de cookie seguros por defecto (`Secure`, `SameSite=Lax` de serie).
* Respuestas ricas: JSON, JSONP, XML, YAML, Markdown, Protocol Buffers, MessagePack, negociación de contenido, streaming y Server-Sent Events.
* Backends de IA: el paquete `ai` renderiza y retransmite por HTTP la salida en streaming de los modelos (deltas de texto, trazas de razonamiento, fragmentos de llamadas a herramientas, uso). No depende de ningún proveedor y no incluye el SDK de ninguno.
* Un cliente HTTP, el paquete `client`, para llamar a otros servicios con las mismas reglas de JSON que usa el lado del servidor.
* Compresión de respuestas: gzip, deflate, brotli, snappy, s2, zstd.
* TLS con Let's Encrypt, y túneles ngrok integrados (opt-in: `import _ "github.com/kataras/iris/v14/core/server/ngrok"`) para poner una aplicación de desarrollo en internet en segundos.
* Pruebas HTTP en memoria: tests de la aplicación completa sin puertos, sin red y sin sleeps.
* Un libro completo que se puede leer online y descargar gratis, un tutorial práctico y un wiki de IA en vivo (mira Aprendiendo Iris más abajo).
* Y lo más importante: obtienes respuestas rápidas y soporte, desde el primer día hasta hoy.

## 📖 Aprendiendo Iris

> ¿Nuevo en Iris? Empieza con el [Tutorial de Inicio](TUTORIAL.md): un solo documento que te lleva de `go mod init` a una API REST probada y con forma de producción. ¿Actualizas desde v12? Mira la [Guía de Migración](MIGRATION.md). ¿Dudas en cualquier punto? Pregunta al **wiki de IA en vivo**, gratuito, en [codewiki.google/github.com/kataras/iris](https://codewiki.google/github.com/kataras/iris), generado por Google a partir del código fuente de Iris y mantenido en sincronía con él, con un chat integrado que responde preguntas sobre el código.

### Instalación

El único requisito es el [lenguaje de programación Go](https://go.dev/dl/), versión 1.27 o posterior.

#### Crea un nuevo proyecto

```sh
$ mkdir myapp
$ cd myapp
$ go mod init myapp
$ go get github.com/kataras/iris/v14@latest # or @v14.0.0
```

<details><summary>Instalar en un proyecto existente</summary>

```sh
$ cd myapp
$ go get github.com/kataras/iris/v14@latest
```

**Ejecutar**

```sh
$ go mod tidy
$ go run .
```

</details>

![](https://www.iris-go.com/images/gifs/install-create-iris.gif)

Iris contiene una **[documentación](https://www.iris-go.com/docs)** extensa y detallada que facilita empezar con el framework.

Para una documentación técnica más detallada puedes visitar nuestros [godocs](https://pkg.go.dev/github.com/kataras/iris/v14@v14.0.0). Para una introducción completa y práctica a la pila recomendada de v14 lee el [Tutorial de Inicio](TUTORIAL.md), y si actualizas desde v12 sigue la [Guía de Migración](MIGRATION.md).

¿Prefieres aprender con código? Los [**59 ejemplos**](_examples/README.md) van desde una primera ruta hasta la inyección de dependencias, la autenticación, las pruebas y los puntos de extensión del framework. Cada uno de ellos se compiló y se ejercitó sobre HTTP antes de subirlo al repositorio.

### El Libro de Iris

<a href="https://iris-go.com/book/"> <img alt="Book cover" src="https://iris-go.com/images/iris-book-cover-sm.jpg?v=14" /> </a>

El libro completo tiene un prefacio, 24 capítulos y un epílogo, con cada ejemplo de código verificado contra el código fuente del framework. Cada capítulo abre con un contexto en lenguaje llano para quienes empiezan y cierra con lecturas adicionales seleccionadas. Todos los capítulos se pueden leer gratis en [iris-go.com/docs](https://iris-go.com/docs/); la [página del libro](https://iris-go.com/book/) tiene el índice completo.

El PDF también es gratis. [Descárgalo directamente](https://iris-go.com/iris-book.pdf): no hay ningún formulario que rellenar ni nada que solicitar. Si el libro te resulta útil, [patrocinar el proyecto](https://iris-go.com/donate/) es la forma de participar en el desarrollo de Iris.

[![follow author on twitter](https://img.shields.io/twitter/follow/makismaropoulos?color=3D8AA3&logoColor=3D8AA3&style=for-the-badge&logo=twitter)](https://twitter.com/intent/follow?screen_name=makismaropoulos)

[![follow Iris web framework on twitter](https://img.shields.io/twitter/follow/iris_framework?color=ee7506&logoColor=ee7506&style=for-the-badge&logo=twitter)](https://twitter.com/intent/follow?screen_name=iris_framework)

[![follow Iris web framework on facebook](https://img.shields.io/badge/Follow%20%40Iris.framework-569-2D88FF.svg?style=for-the-badge&logo=facebook)](https://www.facebook.com/iris.framework)

## 🤖 Herramientas de IA para Iris

Dos herramientas de IA conocen este framework en concreto, más allá de lo que un modelo general sabe sobre Go.

La [wiki de IA en vivo](https://codewiki.google/github.com/kataras/iris) es gratuita y funciona en el navegador: explora la arquitectura o pregunta al chat sobre el código.

[Plexon AI](https://plexon.ai) es un asistente de escritorio para Windows, macOS y Linux, creado por el autor de Iris. Incorpora la skill de Iris ([iris-go.com/ai](https://iris-go.com/ai/)): la API de v14, la cadena del builder, los helpers de `rest`, el mapeo de errores, el SDK de autenticación y los patrones de pruebas, en veintiún documentos de referencia que el asistente carga cuando los necesita mientras escribe tu código. Activa la skill desde el panel de Skills o instala la [persona Software Developer](https://plexon.ai/personas/software-developer/), que la activa junto con el resto de sus herramientas de ingeniería. [Descarga Plexon AI](https://plexon.ai/download/).

Para los patrocinadores, la contribución, la política de seguridad y la licencia, consulta el [README en inglés](README.md).

Este proyecto está licenciado bajo la [licencia BSD 3-clause](LICENSE), igual que el propio proyecto Go.
