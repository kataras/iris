# Examples

This folder provides easy to understand code snippets on how to get started with web development with the Go programming language using the  [Iris](https://github.com/kataras/iris) web framework.

It doesn't contains "best ways" neither explains all its features. It's just a simple, practical cookbook for young Go developers!

## Table of Contents
<!-- when the new book will be ready I should add the link here -->
<img align="right" src="http://iris-go.com/assets/book/cover_4.jpg" width="300" />

* [Level: Beginner](beginner)
    * [Hello World](beginner/hello-world/main.go)
    * [Routes (using httprouter)](beginner/routes-using-httprouter/main.go)
    * [Routes (using gorillamux)](beginner/routes-using-gorillamux/main.go)
    * [Internal Application File Logger](beginner/file-logger/main.go)
    * [Write JSON](beginner/write-json/main.go)
    * [Read JSON](beginner/read-json/main.go)
    * [Read Form](beginner/read-form/main.go)
    * [Favicon](beginner/favicon/main.go)
    * [File Server](beginner/file-server/main.go)
    * [Send Files](beginner/send-files/main.go)
    * [Stream Writer](beginner/stream-writer/main.go)
    * [Listen UNIX Socket](beginner/listen-unix/main.go)
    * [Listen TLS](beginner/listen-tls/main.go)
    * [Listen Letsencrypt (Automatic Certifications)](beginner/listen-letsencrypt/main.go)
* [Level: Intermediate](intermediate)
    * [Send An E-mail](intermediate/e-mail/main.go)
    * [Upload/Read Files](intermediate/upload-files/main.go)
    * [Request Logger](intermediate/request-logger/main.go)
    * [Profiling (pprof)](intermediate/pprof/main.go)
    * [Basic Authentication](intermediate/basicauth/main.go)
    * [HTTP Access Control](intermediate/cors/main.go)
    * [Cache Markdown](intermediate/cache-markdown/main.go)
    * [Localization and Internationalization](intermediate/i18n/main.go)
    * [Recovery](intermediate/recover/main.go)
    * [Graceful Shutdown](intermediate/graceful-shutdown/main.go)
    * [Custom TCP Listener](intermediate/custom-listener/main.go)
    * [Custom HTTP Server](intermediate/custom-httpserver/main.go)
    * [View Engine](intermediate/view)
        * [Overview](intermediate/view/overview/main.go)
        * [Template HTML: Part Zero](intermediate/view/template_html_0/main.go)
        * [Template HTML: Part One](intermediate/view/template_html_1/main.go)
        * [Template HTML: Part Two](intermediate/view/template_html_2/main.go)
        * [Template HTML: Part Three](intermediate/view/template_html_3/main.go)
        * [Template HTML: Part Four](intermediate/view/template_html_4/main.go)
        * [Inject Data Between Handlers](intermediate/view/context-view-data/main.go)
        * [Embedding Templates Into Executable](intermediate/view/embedding-templates-into-app)
        * [Custom Renderer](intermediate/view/custom-renderer/main.go)
    * [Password Hashing](intermediate/password-hashing/main.go)
    * [Sessions](intermediate/sessions)
        * [Overview](intermediate/sessions/overview/main.go)
        * [Encoding & Decoding the Session ID: Secure Cookie](intermediate/sessions/securecookie/main.go)
        * [Standalone](intermediate/sessions/standalone/main.go)
        * [With A Back-End Database](intermediate/sessions/database/main.go)
    * [Flash Messages](intermediate/flash-messages/main.go)
    * [Websockets](intermediate/websockets)
        * [Ridiculous Simple](intermediate/websockets/ridiculous-simple/main.go)
        * [Overview](intermediate/websockets/overview/main.go)
        * [Connection List](intermediate/websockets/connectionlist/main.go)
        * [Native Messages](intermediate/websockets/naive-messages/main.go)
        * [Secure](intermediate/websockets/secure/main.go)
        * [Custom Go Client](intermediate/websockets/custom-go-client/main.go)
* [Level: Advanced](advanced)
    * [Transactions](advanced/transactions/main.go)
    * [HTTP Testing](advanced/httptest/main_test.go)
    * [Watch & Compile Typescript source files](advanced/typescript/main.go)
    * [Cloud Editor](advanced/cloud-editor/main.go)
    * [Online Visitors](advanced/online-visitors/main.go)
    * [URL Shortener using BoltDB](advanced/url-shortener/main.go)
    * [Subdomains](advanced/subdomains)
        * [Single](advanced/subdomains/single/main.go)
        * [Multi](advanced/subdomains/multi/main.go)
        * [Wildcard](advanced/subdomains/wildcard/main.go)



> Don't forget to take a quick look or add your own [examples in the community's repository](https://github.com/iris-contrib/examples)!

> Developers should read the official [documentation](https://godoc.org/gopkg.in/kataras/iris.v6), in depth, for better understanding.
