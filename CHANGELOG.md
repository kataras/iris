## Changelog v1.2.1 -> v2.0.0

Global:

- ```.Templates("./path/*.html")``` -> ``` .Templates().Load("./path/*.html","yourNamespace") //namespace is optionally ```
- ```.TemplateFuncs(...)``` -> ```.Templates().Funcs(...)```
- ```.TemplateDelims("left","right")``` -> ```.Templates().Delims("left","right")```
- ```.GetTemplates()``` -> ```.Templates()```
- ```.Plugin(plugin)``` -> ```.Plugins().Add(plugin)```
- ```.StationOptions``` -> ```IrisConfig```
- ```.Custom``` -> ```.New(...IrisConfig)```
- ```.Listen(...string)``` -> ```.Listen(string)```

Context:
- ```.AddCookie(*fasthttp.Cookie{})``` -> ```.SetCookie(*fasthttp.Cookie{})```
- ```.SetCookie(string,string)``` -> ```.SetCookieKV(string,string)```
- ```.RenderFile(string,interface{}) error``` -> ```.Render(string,interface{}) error```


### Added
- ```IrisConfig { ... MaxRequestBodySize int }```
- ```.Plugins() []Plugin```
- ```Context.RenderNS(namespace string, file string, pageContext interface{}) error```
- ```Context.SetCookieKV(string,string)```


------------

