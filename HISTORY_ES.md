<!-- # History/Changelog <a href="HISTORY_ZH.md"> <img width="20px" src="https://iris-go.com/images/flag-china.svg?v=10" /></a><a href="HISTORY_ID.md"> <img width="20px" src="https://iris-go.com/images/flag-indonesia.svg?v=10" /></a><a href="HISTORY_GR.md"> <img width="20px" src="https://iris-go.com/images/flag-greece.svg?v=10" /></a> -->

# Registro de cambios

### ¿Buscando soporte gratuito y en tiempo real?

    https://github.com/kataras/iris/issues
    https://chat.iris-go.com

### ¿Buscando versiones anteriores?

    https://github.com/kataras/iris/releases

### ¿Quieres ser contratado?

    https://facebook.com/iris.framework

### ¿Debo actualizar mi versión de Iris?

Los desarrolladores no están obligados a actualizar si realmente no lo necesitan. Actualice siempre que se sienta listo.

**Cómo actualizar**: Abra su línea de comandos y ejecute este comando: `go get github.com/kataras/iris@master`.

# Viernes, 16 de agosto 2019 | v11.2.8

- Establecer `Cookie.SameSite` como ` Lax` cuando el uso compartido de sesiones de subdominios esté habilitado[*](https://github.com/kataras/iris/commit/6bbdd3db9139f9038641ce6f00f7b4bab6e62550)
- Agregados y actualizados todos los [Handlers experimentales](https://github.com/kataras/iris/tree/master/_examples/experimental-handlers)
- Nueva función `XMLMap` que envuelve un `map[string]interface{}` y la convierte en un contenido xml válido para representarlo a través del método `Context.XML`
- Se agregaron nuevos campos `ProblemOptions.XML` y ` RenderXML` para renderizar `Problem` como XML(application/problem+xml) en lugar de JSON("application/problem+json) y enriquezca el `Negotiate` para aceptar fácilmente el mime type `application/problem+xml`.

Registro de commits: https://github.com/kataras/iris/compare/v11.2.7...v11.2.8

# Jueves, 15 de agosto 2019 | v11.2.7

Esta versión menor contiene mejoras en los Detalles del problema para las API HTTP implementadas en [v11.2.5](#lunes-12-de-agosto-2019--v1125).

- Ajuste https://github.com/kataras/iris/issues/1335#issuecomment-521319721
- Agregado `ProblemOptions` con `RetryAfter` como se solicitó en: https://github.com/kataras/iris/issues/1335#issuecomment-521330994.
- Agregado alias `iris.JSON` para el tipo de opciones `context#JSON`.

[Ejemplos](https://github.com/kataras/iris/blob/45d7c6fedb5adaef22b9730592255f7bb375e809/_examples/routing/http-errors/main.go#L85) y [wikis](https://github.com/kataras/iris/wiki/Routing-error-handlers#the-problem-type) actualizados.

Referencias:

- https://tools.ietf.org/html/rfc7231#section-7.1.3
- https://tools.ietf.org/html/rfc7807

Registro de commits: https://github.com/kataras/iris/compare/v11.2.6...v11.2.7

# Miércoles, 14 de agosto 2019 | v11.2.6

Permitir [manejar más de una ruta con las mismas rutas y tipos de parámetros pero diferentes funciones de validación de macros](https://github.com/kataras/iris/issues/1058#issuecomment-521110639).

```go
app.Get("/{alias:string regexp(^[a-z0-9]{1,10}\\.xml$)}", PanoXML)
app.Get("/{alias:string regexp(^[a-z0-9]{1,10}$)}", Tour)
```

Registro de commits: https://github.com/kataras/iris/compare/v11.2.5...v11.2.6

# Lunes, 12 de agosto 2019 | v11.2.5

- [Nueva característica: Detalle del problemas para las APIs HTTP](https://github.com/kataras/iris/pull/1336)
- [Agregado Context.AbsoluteURI](https://github.com/kataras/iris/pull/1336/files#diff-15cce7299aae8810bcab9b0bf9a2fdb1R2368)

Registro de commits: https://github.com/kataras/iris/compare/v11.2.4...v11.2.5

# Viernes, 09 de agosto 2019 | v11.2.4

- Ajustes [iris.Jet: no view engine found for '.jet' or '.html'](https://github.com/kataras/iris/issues/1327)
- Ajustes [ctx.ViewData no funciona con JetEngine](https://github.com/kataras/iris/issues/1330)
- **Nueva característica**: [Override de métodos HTTP](https://github.com/kataras/iris/issues/1325)
- Ajustes [Bajo rendimiento en session.UpdateExpiration en más de 200 mil keys con nueva librería radix](https://github.com/kataras/iris/issues/1328) al introducir el campo de configuración `sessions.Config.Driver` que se establece de forma predeterminada en `Redigo()` pero también se puede establecer en  `Radix()`, futuras adiciones son bienvenidas.

Registro de commits: https://github.com/kataras/iris/compare/v11.2.3...v11.2.4

# Martes, 30 de julio 2019 | v11.2.3

- [Nueva característica: Manejar diferentes tipos de parámetros en la misma ruta](https://github.com/kataras/iris/issues/1315)
- [Nueva característica: Negociación de contenido](https://github.com/kataras/iris/issues/1319)
- [Context.ReadYAML](https://github.com/kataras/iris/tree/master/_examples/http_request/read-yaml)
- Ajustes https://github.com/kataras/neffos/issues/1#issuecomment-515698536

# Miércoles, 24 de julio 2019 | v11.2.2

Sesiones como middleware:

```go
import "github.com/kataras/iris/v12/sessions"
// [...]

app := iris.New()
sess := sessions.New(sessions.Config{...})

app.Get("/path", func(ctx iris.Context){
    session := sessions.Get(ctx)
    // [work with session...]
})
```

- Agregado `Session.Len() int` para devolver el número total de valores/entradas almacenados.
- Permitir que `Context.HTML` y `Context.Text` acepten tambien un argumento `args ...interface{}` opcional y variable.

## v11.1.1

- https://github.com/kataras/iris/issues/1298
- https://github.com/kataras/iris/issues/1207

# Martes, 23 de julio 2019 | v11.2.0

Lea sobre la nueva versión liberada en: https://www.facebook.com/iris.framework/posts/3276606095684693
