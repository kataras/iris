# Examples

This folder provides easy to understand code snippets on how to get started with web development with the Go programming language using the  [Iris](https://github.com/kataras/iris) web framework.


It doesn't contains "best ways" neither explains all its features. It's just a simple, practical cookbook for young Go developers!

Developers should read the official [documentation](https://godoc.org/gopkg.in/kataras/iris.v6) in depth.


<a href ="https://github.com/kataras/iris"> <img align="right" src="http://iris-go.com/assets/book/cover_4.jpg" width="300" /> </a>


## Table of Contents

* [Level: Beginner](examples/beginner)
    * [Hello World](examples/beginner/hello-world/main.go)
    * [Routes (using httprouter)](examples/beginner/routes-using-httprouter/main.go)
    * [Routes (using gorillamux)](examples/beginner/routes-using-gorillamux/main.go)
    * [Write JSON](examples/beginner/write-json/main.go)
    * [Read JSON](examples/beginner/read-json/main.go)
    * [Read Form](examples/beginner/read-form/main.go)
    * [Favicon](examples/beginner/favicon/main.go)
    * [File Server](examples/beginner/file-server/main.go)
    * [Send Files](examples/beginner/send-files/main.go)
* [Level: Intermediate](examples/intermediate)
    * [Send An E-mail](examples/intermediate/e-mail/main.go)
    * [Upload/Read Files](examples/intermediate/upload-files/main.go)
    * [Request Logger](examples/intermediate/request-logger/main.go)
    * [Profiling (pprof)](examples/intermediate/pprof/main.go)
    * [Basic Authentication](examples/intermediate/basicauth/main.go)
    * [HTTP Access Control](examples/intermediate/cors/main.go)
    * [Cache Markdown](examples/intermediate/cache-markdown/main.go)
    * [Localization and Internationalization](examples/intermediate/i18n/main.go)
    * [Recovery](examples/intermediate/recover/main.go)
    * [Graceful Shutdown](examples/intermediate/graceful-shutdown/main.go)
    * [View Engine](examples/intermediate/view)
        * [Overview](examples/intermediate/view/overview/main.go)
        * [Embedding Templates Into Executable](examples/intermediate/embedding-templates-into-app)
        * [Template HTML: Part Zero](examples/intermediate/view/template_html_0/main.go)
        * [Template HTML: Part One](examples/intermediate/view/template_html_1/main.go)
        * [Template HTML: Part Two](examples/intermediate/view/template_html_2/main.go)
        * [Template HTML: Part Three](examples/intermediate/view/template_html_3/main.go)
        * [Template HTML: Part Four](examples/intermediate/view/template_html_4/main.go)
        * [Custom Renderer](examples/intermediate/view/custom-renderer/main.go)
    * [Password Hashing](examples/intermediate/password-hashing/main.go)
    * [Sessions](examples/intermediate/sessions)
        * [Overview](examples/intermediate/sessions/overview/main.go)
        * [Encoding & Decoding the Session ID: Secure Cookie](examples/intermediate/sessions/securecookie/main.go)
        * [Standalone](examples/intermediate/sessions/standalone/main.go)
        * [With A Back-End Database](examples/intermediate/sessions/database/main.go)
    * [Flash Messages](examples/intermediate/flash-messages/main.go)
    * [Websockets](examples/intermediate/websockets)
        * [Ridiculous Simple](examples/intermediate/websockets/ridiculous-simple/main.go)
        * [Overview](examples/intermediate/websockets/overview/main.go)
        * [Connection List](examples/intermediate/websockets/connectionlist/main.go)
        * [Native Messages](examples/intermediate/websockets/naive-messages/main.go)
        * [Secure](examples/intermediate/websockets/secure/main.go)
        * [Custom Go Client](examples/intermediate/websockets/custom-go-client/main.go)
* [Level: Advanced](examples/advanced)
    * [HTTP Testing](examples/advanced/httptest/main_test.go)
    * [Watch & Compile Typescript source files](examples/advanced/typescript/main.go)
    * [Cloud Editor](examples/advanced/cloud-editor/main.go)
    * [Online Visitors](examples/advanced/online-visitors/main.go)
    * [URL Shortener using BoltDB](examples/advanced/url-shortener/main.go)

> Take look at the [community examples](https://github.com/iris-contrib/examples) too!
