## Changelog v1.2.1 -> v2.0.0

Global:
- - ```Context.RenderFile(file string, pageContext interface{}) error``` -> - ```Context.Render(file string, pageContext interface{}) error```
- ```.Templates("./path/*.html")``` -> see iris.RenderConfig
- ```.TemplateFuncs(...)``` -> see iris.RenderConfig
- ```.TemplateDelims("left","right")``` -> see iris.RenderConfig
- ```.GetTemplates()``` -> removed
- ```.Plugin(plugin)``` -> ```.Plugins().Add(plugin)```
- ```.StationOptions``` -> ```IrisConfig```
- ```.Custom``` -> ```.New(...IrisConfig)```
- ```.Listen(...string)``` -> ```.Listen(string)```
- ```StationOptions``` -> ```*IrisConfig```

> IrisConfig and RenderConfig are passed by reference now

Context:
- ```.AddCookie(*fasthttp.Cookie{})``` -> ```.SetCookie(*fasthttp.Cookie{})```
- ```.SetCookie(string,string)``` -> ```.SetCookieKV(string,string)```
- ```Context.Render("mypage.html",mypage{})``` -> ```Context.Render("mypage",mypage{},"otherThanConfigLayout")```

Render Context's removed:
- ```Context.HTML```
- ```Context.WriteData```
- ```Context.WriteText```
- ```Context.WriteJSON```
- ```Context.WriteXML```
- ```Context.RenderData```
- ```Context.RenderJSON```
- ```Context.RenderXML```
- all these are replaced with ```Context.HTML, Context.Data, Context.Text, Context.JSON, Context.XML```

### Added
- ```IrisConfig { ... MaxRequestBodySize int }```  MaxRequestBodySize is the only options that can be changed after server listen - using SetMaxRequestBodySize(int)
- ```IrisConfig {...Render: RenderConfig}```  Render can be changed after declaration but before server's listen - using SetRenderConfig(RenderConfig)
- ```.Plugins() []Plugin```
- ```Context.SetCookieKV(string,string)```
- ```JSONP(status int, callback string, v interface{}) error```

- ```RenderConfig``` defaults:

```go
iris.New(&iris.Config{
Render: &iris.RenderConfig{
    Directory: "templates",
    Asset: nil,
    AssetNames: nil,
    Layout: "",
    Extensions: []string{".html"},
    Funcs: []template.FuncMap{},
    Delims: iris.Delims{"{{", "}}"},
    Charset: "UTF-8",
    IndentJSON: false,
    IndentXML: false,
    PrefixJSON: []byte(""),
    PrefixXML: []byte(""),
    HTMLContentType: "text/html",
    IsDevelopment: false,
    UnEscapeHTML: false,
    StreamingJSON: false,
    RequirePartials: false,
    DisableHTTPErrorRendering: false,
},})


```
### Types

- Station -> Iris
- StationOptions -> Iris.IrisConfig [IrisConfig and not Config because of the iris.Config() func]
- RouterDomain -> Router.serveDomainFunc
- All string and file utils moved to ```utils``` package
- HTTPErrors -> HTTPErrorContainer
- iris.Error -> errors.Error
- iris.Server -> server.Server
- iris.Server.ServerOptions -> server.Config
- much more but are useless for you...


------------

