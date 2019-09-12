<div dir="rtl" align='right' >
<!-- # Iris Web Framework <a href="README_ZH.md"> <img width="20px" src="https://iris-go.com/images/flag-china.svg?v=10" /></a> <a href="README_RU.md"><img width="20px" src="https://iris-go.com/images/flag-russia.svg?v=10" /></a> <a href="README_ID.md"> <img width="20px" src="https://iris-go.com/images/flag-indonesia.svg?v=10" /></a> <a href="README_GR.md"><img width="20px" src="https://iris-go.com/images/flag-greece.svg?v=10" /></a> <a href="README_PT_BR.md"><img width="20px" src="https://iris-go.com/images/flag-pt-br.svg?v=10" /></a> <a href="README_JPN.md"><img width="20px" src="https://iris-go.com/images/flag-japan.svg?v=10" /></a> -->

# آیریس <a href="README_ZH.md"><img width="20px" src="https://iris-go.com/images/flag-china.svg?v=10" /></a> <a href="README_GR.md"><img width="20px" src="https://iris-go.com/images/flag-greece.svg?v=10" /></a> <a href="README_ES.md"><img width="20px" src="https://iris-go.com/images/flag-spain.png" /></a> <a href="README_KO.md"><img width="20px" src="https://upload.wikimedia.org/wikipedia/commons/0/09/Flag_of_South_Korea.svg" />
<div dir="ltr" align='left' >

[![build status](https://img.shields.io/travis/kataras/iris/master.svg?style=for-the-badge)](https://travis-ci.org/kataras/iris) [![report card](https://img.shields.io/badge/report%20card-a%2B-ff3333.svg?style=for-the-badge)](https://goreportcard.com/report/github.com/kataras/iris)<!--[![godocs](https://img.shields.io/badge/go-%20docs-488AC7.svg?style=for-the-badge)](https://godoc.org/github.com/kataras/iris)--> [![view examples](https://img.shields.io/badge/learn%20by-examples-0077b3.svg?style=for-the-badge)](https://github.com/kataras/iris/tree/master/_examples) [![chat](https://img.shields.io/gitter/room/iris_go/community.svg?color=blue&logo=gitter&style=for-the-badge)](https://gitter.im/iris_go/community) [![release](https://img.shields.io/badge/release%20-v11.2-0077b3.svg?style=for-the-badge)](https://github.com/kataras/iris/releases)
</div>

آیریس سریع ترین و ساده ترین و موثرترین فریمورک وب در زبان GO میباشد. آیریس ساختاری بسیار زیبا و کارآمد را فراهم کرده است تا شما از آن برای پروژه های بعدی تان استفاده کنید. .

برای این که بدانید دیگران در مورد آیریس چه می گویند لطفا در این لینک کلیک کنید [دیگران در مورد آیریس چه می گویند](https://iris-go.com/testimonials/) لطفا این پروژه را در گیتاب **استار** کنید.

<div dir="ltr" align="left">

> نسخه 11.2 **آماده شد**

[![https://www.facebook.com/iris.framework/posts/3276606095684693](https://iris-go.com/images/iris-112-released.png)](https://www.facebook.com/iris.framework/posts/3276606095684693)
</div>

## آموزش آیریس

<details>
<summary>شروع سریع</summary>

<div dir="ltr" align="left">

<div dir="rtl" align="right">

```sh

# فرض کنید همچین کدی را در فایل example.go نوشته اید 
```
</div>

```sh
$ cat example.go
```

```go
package main

import "github.com/kataras/iris"

func main() {
    app := iris.Default()
    app.Get("/ping", func(ctx iris.Context) {
        ctx.JSON(iris.Map{
            "message": "pong",
        })
    })

    app.Run(iris.Addr(":8080"))
}
```

```sh
# run example.go and
# visit http://localhost:8080/ping on browser
$ go run example.go
```
<div>


> ایریس از پروژه ی [muxie](https://github.com/kataras/muxie) که موثرترین و سریع ترین پروژه مسیریابی در GO می باشد استفاده می کند.


</details>

آیریس داری **[wiki](https://github.com/kataras/iris/wiki)** بسیار کامل و گسترده ای میباشد که یادگیری ان را ساده می کند.


شما برای مشاهده و خواندن داکیومنت های فنی میتوانید به [godocs](https://godoc.org/github.com/kataras/iris) مراجعه کنید و همچنین برای مشاهده مثال ها و کد های قابل اجرا همیشه میتوانید از [_examples](_examples/) repository's subdirectory. استفادهده کنید . 


### آیا شما مطالعه کردن در طول سفر را دوست دارید ؟

<div dir="ltr" align="left">

<a href="https://bit.ly/iris-req-book"> <img alt="Book cover" src="https://iris-go.com/images/iris-book-cover-sm.jpg" width="200" /> </a>

</div>

شما می توانید درخواست یک نسخه PDF آن را از  اینجا [درخواست](https://bit.ly/iris-req-book) بدهید . رایگان.

## Contributing

We'd love to see your contribution to the Iris Web Framework! For more information about contributing to the Iris project please check the [CONTRIBUTING.md](CONTRIBUTING.md) file.

[List of all Contributors](https://github.com/kataras/iris/graphs/contributors)

## Security Vulnerabilities

If you discover a security vulnerability within Iris, please send an e-mail to [iris-go@outlook.com](mailto:iris-go@outlook.com). All security vulnerabilities will be promptly addressed.

## License

The project name "Iris" was inspired by the Greek mythology.

Iris Web Framework is free and open-source software licensed under the [3-Clause BSD License](LICENSE).

<div>