# Iris Web Framework <a href="README.md"> <img width="20px" src="https://iris-go.com/images/flag-unitedkingdom.svg?v=10" /> </a> <a href="README_ZH.md"> <img width="20px" src="https://iris-go.com/images/flag-china.svg?v=10" /> </a> <a href="README_RU.md"><img width="20px" src="https://iris-go.com/images/flag-russia.svg?v=10" /></a> <a href="README_GR.md"><img width="20px" src="https://iris-go.com/images/flag-greece.svg?v=10" /></a> <a href="README_PT_BR.md"><img width="20px" src="https://iris-go.com/images/flag-pt-br.svg?v=10" /></a> <a href="README_JPN.md"><img width="20px" src="https://iris-go.com/images/flag-japan.svg?v=10" /></a>

<a href="https://iris-go.com"> <img align="right" width="169px" src="https://iris-go.com/images/icon.svg?v=a" title="logo created by @merry.dii" /> </a>

[![build status](https://img.shields.io/travis/kataras/iris/master.svg?style=flat-square)](https://travis-ci.org/kataras/iris)<!-- [![release](https://img.shields.io/github/release/kataras/iris.svg?style=flat-square)](https://github.com/kataras/iris/releases)--> [![report card](https://img.shields.io/badge/report%20card-a%2B-ff3333.svg?style=flat-square)](http://goreportcard.com/report/kataras/iris) [![vscode-iris](https://img.shields.io/badge/ext%20-vscode-0c77e3.svg?style=flat-square)](https://marketplace.visualstudio.com/items?itemName=kataras2006.iris)<!--[![github closed issues](https://img.shields.io/github/issues-closed-raw/kataras/iris.svg?style=flat-square)](https://github.com/kataras/iris/issues?q=is%3Aissue+is%3Aclosed)--> [![chat](https://img.shields.io/badge/community-%20chat-00BCD4.svg?style=flat-square)](https://kataras.rocket.chat/channel/iris) [![view examples](https://img.shields.io/badge/learn%20by-examples-0077b3.svg?style=flat-square)](https://github.com/kataras/iris/tree/master/_examples/routing) [![release](https://img.shields.io/badge/release%20-v11.1-0077b3.svg?style=flat-square)](https://github.com/kataras/iris/releases)

Iris adalah web framework yang cepat, sederhana namun berfitur lengkap dan sangat efisien untuk Go.

Iris menyediakan fondasi yang indah expresif dan mudah digunakan untuk website atau API anda selanjutnya.

Akhirnya, framework nyata yang setara dengan expressjs untuk Go Programming Language.

Pelajari apa yang [orang lain katakan tentang Iris](#support) dan [star](https://github.com/kataras/iris/stargazers) github repository ini untuk [mendapatkan informasi terbaru](https://facebook.com/iris.framework).

## Donatur

Terima kasih kepada seluruh donatur kami! 🙏 [Menjadi donatur](https://iris-go.com/donate)

<a href="https://iris-go.com/donate" target="_blank"><img src="https://iris-go.com/backers.svg?v=2"/></a>

```sh
$ cat example.go
```

```go
package main

import "github.com/kataras/iris"

func main() {
    app := iris.New()
    // Memuat semua template dari folder "./views"
    // yang memiliki ekstensi ".html" dan menguraikannya
    // menggunakan package standard `html/template`.
    app.RegisterView(iris.HTML("./views", ".html"))

    // Method:    GET
    // Resource:  http://localhost:8080
    app.Get("/", func(ctx iris.Context) {
        // Bind: {{.message}} with "Hello world!"
        ctx.ViewData("message", "Hello world!")
        // Render template file: ./views/hello.html
        ctx.View("hello.html")
    })

    // Method:    GET
    // Resource:  http://localhost:8080/user/42
    //
    // Butuh menggunakan custom regexp sebagai gantinya?
    // Mudah,
    // cukup tandai tipe parameter menjadi 'string'
    // yang akan menerima semua dan akan menggunakan
    // fungsi macro `regexp`, Contoh:
    // app.Get("/user/{id:string regexp(^[0-9]+$)}")
    app.Get("/user/{id:long}", func(ctx iris.Context) {
        userID, _ := ctx.Params().GetInt64("id")
        ctx.Writef("User ID: %d", userID)
    })

    // Menyalakan server menggunakan network address.
    app.Run(iris.Addr(":8080"))
}
```

> Pelajari lebih lanjut tentang tipe parameter di path dengan klik [disini](_examples/routing/dynamic-path/main.go#L31)

```html
<!-- file: ./views/hello.html -->
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

## Instalasi

Satu - satunya persyaratan adalah [Go Programming Language](https://golang.org/dl/)

```sh
$ go get -u github.com/kataras/iris
```

Iris mengambil keuntungan dari fitur [vendor directory](https://docs.google.com/document/d/1Bz5-UB7g2uPBdOx-rw5t9MxJwkfpx90cqG9AFL0JAYo). Anda mendapatkan build yang benar - benar dapat direproduksi, karena metode ini menjaga terhadap penggantian nama dan penghapusan di upstream.

[![Iris vs .NET Core(C#) vs Node.js (Express)](https://iris-go.com/images/benchmark-new-gray.png)](_benchmarks/README_UNIX.md)

_Diperbarui pada: [Tuesday, 21 November 2017](_benchmarks/README_UNIX.md)_

<details>
<summary>Benchmarks dari sumber pihak ketiga terhadap rest web frameworks</summary>

![Perbandingan dengan framework lain](https://raw.githubusercontent.com/smallnest/go-web-framework-benchmark/4db507a22c964c9bc9774c5b31afdc199a0fe8b7/benchmark.png)

</details>

## Dukungan

- File [HISTORY](HISTORY_ID.md#su-18-november-2018--v1110) adalah sahabat anda, file tersebut memiliki informasi terkait fitur dan perubahan terbaru
- Apakah anda menemukan bug? Laporkan itu melalui [github issues](https://github.com/kataras/iris/issues)
- Apakah anda memiliki pertanyaan atau butuh untuk bicara kepada seseorang yang sudah berpengalaman untuk menyelesaikan masalah secara langsung? Gabung bersama kami di [community chat](https://chat.iris-go.com)
- Lengkapi laporan user-experience berbasis formulir kami dengan tekan [disini](https://docs.google.com/forms/d/e/1FAIpQLSdCxZXPANg_xHWil4kVAdhmh7EBBHQZ_4_xSZVDL-oCC_z5pA/viewform?usp=sf_link)
- Apakah anda menyukai framework ini? Tweet sesuatu tentang ini! Orang - orang yang sudah berbicara:

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

Untuk informasi lebih lanjut mengenai kontribusi terhadap project Iris, mohon untuk mengecek file [CONTRIBUTING.md](CONTRIBUTING.md).

[Daftar seluruh Kontributor](https://github.com/kataras/iris/graphs/contributors)

## Belajar

Pertama - tama, cara yang paling tepat untuk memulai dengan web framework adalah dengan mempelajari dasar dari bahasa pemrograman dan kemampuan dasar `http`, apabila aplikasi web anda adalah proyek pribadi yang sangat sederhana tanpa kebutuhan kinerja dan pemeliharaan, anda dapat melanjutkan hanya dengan standard packages. Setelah itu, ikut petunjuknya:

- Kunjungi **100+1** **[contoh](_examples)** dan beberapa [iris starter kits](#iris-starter-kits) yang kami buat untuk anda
- Baca [godocs](https://godoc.org/github.com/kataras/iris) untuk penjelasan yang lebih detail
- Siapkan secangkir kopi atau teh, apapun yang paling menyenangkan anda, dan baca beberapa [artikel](#articles) yang kami temukan untuk anda

### Iris starter kits

<!-- table form 
| Description | Link |
| -----------|-------------|
| Hasura hub starter project with a ready to deploy golang helloworld webapp with IRIS! | https://hasura.io/hub/project/hasura/hello-golang-iris |
| A basic web app built in Iris for Go |https://github.com/gauravtiwari/go_iris_app |
| A mini social-network created with the awesome Iris💖💖 | https://github.com/iris-contrib/Iris-Mini-Social-Network |
| Iris isomorphic react/hot reloadable/redux/css-modules starter kit | https://github.com/iris-contrib/iris-starter-kit |
| Demo project with react using typescript and Iris | https://github.com/ionutvilie/react-ts |
| Self-hosted Localization Management Platform built with Iris and Angular | https://github.com/iris-contrib/parrot |
| Iris + Docker and Kubernetes | https://github.com/iris-contrib/cloud-native-go |
| Quickstart for Iris with Nanobox | https://guides.nanobox.io/golang/iris/from-scratch |
-->

1. [A basic CRUD API in golang with Iris](https://github.com/jebzmos4/Iris-golang)
2. [A basic web app built in Iris for Go](https://github.com/gauravtiwari/go_iris_app)
3. [A mini social-network created with the awesome Iris💖💖](https://github.com/iris-contrib/Iris-Mini-Social-Network)
4. [Iris isomorphic react/hot reloadable/redux/css-modules starter kit](https://github.com/iris-contrib/iris-starter-kit)
5. [Demo project with react using typescript and Iris](https://github.com/ionutvilie/react-ts)
6. [Self-hosted Localization Management Platform built with Iris and Angular](https://github.com/iris-contrib/parrot)
7. [Iris + Docker and Kubernetes](https://github.com/iris-contrib/cloud-native-go)
8. [Quickstart for Iris with Nanobox](https://guides.nanobox.io/golang/iris/from-scratch)
9. [A Hasura starter project with a ready to deploy Golang hello-world web app with IRIS](https://hasura.io/hub/project/hasura/hello-golang-iris)

> Apakah anda membuat hal yang serupa? [Beritahu kami](https://github.com/kataras/iris/pulls)!

### Middleware

Iris memiliki koleksi handler yang hebat[[1]](middleware/)[[2]](https://github.com/iris-contrib/middleware) yang dapat anda gunakan berdampingan dengan aplikasi web anda. Namun, anda tidak terbatas oleh itu saja - anda bebas menggunakan third-party middleware yang compatible dengan package [net/http](https://golang.org/pkg/net/http/), [_examples/convert-handlers](_examples/convert-handlers) akan menunjukkan caranya.

Iris, tidak seperti yang lain, 100% compatible dengan standards dan maka dari itu mayoritas dari perusahaan besar yang mengadaptasi Go kepada alur kerja mereka, seperti Jaringan Telivisi yang sangat terkenal di US, mempercayai Iris; framework yang up-to-date dan ini akan selalu selaras dengan package std `net/http` yang dimodernisasi oleh Pencipta Go di setiap release dari Go Programming Language.

### Articles

* [CRUD REST API in Iris (a framework for golang)](https://medium.com/@jebzmos4/crud-rest-api-in-iris-a-framework-for-golang-a5d33652401e)
* [A Todo MVC Application using Iris and Vue.js](https://hackernoon.com/a-todo-mvc-application-using-iris-and-vue-js-5019ff870064)
* [A Hasura starter project with a ready to deploy Golang hello-world web app with IRIS](https://bit.ly/2lmKaAZ)
* [Top 6 web frameworks for Go as of 2017](https://blog.usejournal.com/top-6-web-frameworks-for-go-as-of-2017-23270e059c4b)
* [Iris Go Framework + MongoDB](https://medium.com/go-language/iris-go-framework-mongodb-552e349eab9c)
* [How to build a file upload form using DropzoneJS and Go](https://hackernoon.com/how-to-build-a-file-upload-form-using-dropzonejs-and-go-8fb9f258a991)
* [How to display existing files on server using DropzoneJS and Go](https://hackernoon.com/how-to-display-existing-files-on-server-using-dropzonejs-and-go-53e24b57ba19)
* [Iris, a modular web framework](https://medium.com/@corebreaker/iris-web-cd684b4685c7)
* [Go vs .NET Core in terms of HTTP performance](https://medium.com/@kataras/go-vs-net-core-in-terms-of-http-performance-7535a61b67b8)
* [Iris Go vs .NET Core Kestrel in terms of HTTP performance](https://hackernoon.com/iris-go-vs-net-core-kestrel-in-terms-of-http-performance-806195dc93d5)
* [How to Turn an Android Device into a Web Server](https://twitter.com/ThePracticalDev/status/892022594031017988)
* [Deploying a Iris Golang app in hasura](https://medium.com/@HasuraHQ/deploy-an-iris-golang-app-with-backend-apis-in-minutes-25a559bf530b)
* [A URL Shortener Service using Go, Iris and Bolt](https://medium.com/@kataras/a-url-shortener-service-using-go-iris-and-bolt-4182f0b00ae7)

### Video Courses

* [Daily Coding - Web Framework Golang: Iris Framework]( https://www.youtube.com/watch?v=BmOLFQ29J3s) by WarnabiruTV, sumber: youtube, biaya: **GRATIS**
* [Tutorial Golang MVC dengan Iris Framework & Mongo DB](https://www.youtube.com/watch?v=uXiNYhJqh2I&list=PLMrwI6jIZn-1tzskocnh1pptKhVmWdcbS) (19 parts so far) by Musobar Media, sumber: youtube, biaya: **GRATIS**
* [Go/Golang 27 - Iris framework : Routage de base](https://www.youtube.com/watch?v=rQxRoN6ub78) by stephgdesign, sumber: youtube, biaya: **GRATIS**
* [Go/Golang 28 - Iris framework : Templating](https://www.youtube.com/watch?v=nOKYV073S2Y) by stephgdesignn, sumber: youtube, biaya: **GRATIS**
* [Go/Golang 29 - Iris framework : Paramètres](https://www.youtube.com/watch?v=K2FsprfXs1E) by stephgdesign, sumber: youtube, biaya: **GRATIS**
* [Go/Golang 30 - Iris framework : Les middelwares](https://www.youtube.com/watch?v=BLPy1So6bhE) by stephgdesign, sumber: youtube, biaya: **GRATIS**
* [Go/Golang 31 - Iris framework : Les sessions](https://www.youtube.com/watch?v=RnBwUrwgEZ8) by stephgdesign, sumber: youtube, biaya: **GRATIS**

### Get hired

Ada beberapa perusahaan dan start-up yang mencari web developer Go yang memiliki pengalaman menggunakn Iris, kami mencarikan untuk anda setiap hari dan kami post informasi tersebut melalui [facebook page](https://www.facebook.com/iris.framework) kami, like page kami untuk mendapatkan notifikasi, kami sudah mempost beberapa dari mereka.

## License

Iris dilisensikan di bawah [3-Clause BSD License](LICENSE). Iris 100% gratis dan software open-source.

Apabila ada pertanyaan mengenai lisensi, anda dapat mengirimkan [e-mail](mailto:kataras2006@hotmail.com?subject=Iris%20License).
