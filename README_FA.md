# Iris Web Framework

<div dir="ltr" align='left' >

[![build status](https://img.shields.io/travis/kataras/iris/master.svg?style=for-the-badge&logo=travis)](https://travis-ci.org/kataras/iris) [![FOSSA Status](https://img.shields.io/badge/LICENSE%20SCAN-PASSING❤️-CD2956?style=for-the-badge&logo=fossa)](https://app.fossa.io/projects/git%2Bgithub.com%2Fkataras%2Firis?ref=badge_shield)<!--[![report card](https://img.shields.io/badge/report%20card-a%2B-ff3333.svg?style=for-the-badge)](https://goreportcard.com/report/github.com/kataras/iris)--><!--[![godocs](https://img.shields.io/badge/go-%20docs-488AC7.svg?style=for-the-badge)](https://pkg.go.dev/github.com/kataras/iris/v12@v12.2.0)--> [![view examples](https://img.shields.io/badge/learn%20by-examples-0C8EC5.svg?style=for-the-badge&logo=go)](https://github.com/kataras/iris/tree/master/_examples) [![chat](https://img.shields.io/gitter/room/iris_go/community.svg?color=7E18DD&logo=gitter&style=for-the-badge)](https://gitter.im/iris_go/community)<!--[![donate on PayPal](https://img.shields.io/badge/support-PayPal-blue.svg?style=for-the-badge)](https://iris-go.com/donate)--><!-- [![release](https://img.shields.io/badge/release%20-v12.0-0077b3.svg?style=for-the-badge)](https://github.com/kataras/iris/releases) -->

</div>

آیریس سریع ترین و ساده ترین و موثرترین فریمورک وب در زبان GO میباشد. آیریس ساختاری بسیار زیبا و کارآمد را فراهم کرده است تا شما از آن برای پروژه های بعدی تان استفاده کنید. .

برای این که بدانید دیگران در مورد آیریس چه می گویند لطفا در این لینک کلیک کنید [دیگران در مورد آیریس چه می گویند](https://iris-go.com/testimonials/) لطفا این پروژه را در گیتاب **استار** کنید.

[![](https://media.giphy.com/media/j5WLmtvwn98VPrm7li/giphy.gif)](https://iris-go.com/testimonials/)

[![Benchmarks: Apr 2, 2020 at 12:13pm (UTC)](https://iris-go.com/images/benchmarks.svg)](https://github.com/kataras/server-benchmarks)

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

import "github.com/kataras/iris/v12"

func main() {
    app := iris.Default()
    app.Get("/ping", func(ctx iris.Context) {
        ctx.JSON(iris.Map{
            "message": "pong",
        })
    })

    app.Listen(":8080")
}
```

```sh
# run example.go and
# visit http://localhost:8080/ping on browser
$ go run example.go
```

<div>

<div dir="rtl" align="right" >

> ایریس از پروژه ی [muxie](https://github.com/kataras/muxie) که موثرترین و سریع ترین پروژه مسیریابی در GO می باشد استفاده می کند.

<div>

</details>

آیریس داری **[wiki](https://github.com/kataras/iris/wiki)** بسیار کامل و گسترده ای میباشد که یادگیری ان را ساده می کند.

شما برای مشاهده و خواندن داکیومنت های فنی میتوانید به [godocs](https://pkg.go.dev/github.com/kataras/iris/v12@v12.2.0) مراجعه کنید و همچنین برای مشاهده مثال ها و کد های قابل اجرا همیشه میتوانید از [مثال ها](_examples/) استفاده کنید .

### آیا شما مطالعه کردن در طول سفر را دوست دارید ؟

<div dir="ltr" align="left">

<a href="https://iris-go.com/#book"> <img alt="Book cover" src="https://iris-go.com/images/iris-book-cover-sm.jpg?v=12" /> </a>

</div>

[![follow author](https://img.shields.io/twitter/follow/makismaropoulos.svg?style=for-the-badge)](https://twitter.com/intent/follow?screen_name=makismaropoulos)

شما میتوانید در خواست یک نسخه PDF داکیومنت ر ا به صورت رایگان از اینجا بدهید [درخواست](https://bit.ly/iris-req-book)

## مشارکت کردن

ما دوست داریم که شما در فریمورک آیریس مشارکت کنید و کد ها را توسعه و بهبود ببخشید ! برای اطلاع بیشتر در مورد نحوه ی مشارکت کردن در این پروژه لطفا اینجا را بررسی کنید [CONTRIBUTING.md](CONTRIBUTING.md)

[مشاهده ی همه ی مشارکت کننده ها](https://github.com/kataras/iris/graphs/contributors)

## باگ های امنیتی

اگر شما باگ های امنیتی در آیریس پیدا کردید لطفا یک ایمیل به [iris-go@outlook.com](mailto:iris-go@outlook.com) ارسال کنید. همه ی باگ های امنیتی بلافاصله برطرف میشود.

## مجوز

نام پروژه آیریس ریشه ای یونانی دارد.

فریمورک آیریس رایگان و سورس باز و تحت مجوز [3-Clause BSD License](LICENSE) می باشد.

<div>
