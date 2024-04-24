<div dir="rtl">

## خبرها
    
> این شاخه تحت توسعه است. برای رفتن به شاخه نسخه بعدی [v12.2.0](HISTORY.md#Next) یا اگر به دنبال یک انتشار پایدار هستید, به جای آن به شاخه [v12.1.8 branch](https://github.com/kataras/iris/tree/v12.1.8) مراجعه کنید.
    
> ![](https://iris-go.com/images/cli.png) همین امروز برنامه رسمی [Iris Command Line Interface](https://github.com/kataras/iris-cli) را امتحان کنید.

> با توجه به بالا بودن حجم کار، ممکن است در پاسخ به [سوالات](https://github.com/kataras/iris/issues) شما تاخیری وجود داشته باشد.

# Iris Web Framework
    
[![build status](https://img.shields.io/github/actions/workflow/status/kataras/iris/ci.yml?branch=main&style=for-the-badge)](https://github.com/kataras/iris/actions/workflows/ci.yml) [![FOSSA Status](https://img.shields.io/badge/LICENSE%20SCAN-PASSING❤️-CD2956?style=for-the-badge&logo=fossa)](https://app.fossa.io/projects/git%2Bgithub.com%2Fkataras%2Firis?ref=badge_shield)<!--[![report card](https://img.shields.io/badge/report%20card-a%2B-ff3333.svg?style=for-the-badge)](https://goreportcard.com/report/github.com/kataras/iris)--><!--[![godocs](https://img.shields.io/badge/go-%20docs-488AC7.svg?style=for-the-badge)](https://pkg.go.dev/github.com/kataras/iris/v12@v12.2.11)--> [![view examples](https://img.shields.io/badge/learn%20by-examples-0C8EC5.svg?style=for-the-badge&logo=go)](https://github.com/kataras/iris/tree/main/_examples) [![chat](https://img.shields.io/gitter/room/iris_go/community.svg?color=7E18DD&logo=gitter&style=for-the-badge)](https://gitter.im/iris_go/community)<!--[![donate on PayPal](https://img.shields.io/badge/support-PayPal-blue.svg?style=for-the-badge)](https://iris-go.com/donate)--><!-- [![release](https://img.shields.io/badge/release%20-v12.0-0077b3.svg?style=for-the-badge)](https://github.com/kataras/iris/releases) -->
    
آیریس یک چارچوب وب پر سرعت ، ساده و در عین حال کاملاً برجسته و بسیار کارآمد برای Go است.
</div>

<details><summary>Simple Handler</summary>

```go
package main

import "github.com/kataras/iris/v12"

type (
  request struct {
    Firstname string `json:"firstname"`
    Lastname  string `json:"lastname"`
  }

  response struct {
    ID      uint64 `json:"id"`
    Message string `json:"message"`
  }
)

func main() {
  app := iris.New()
  app.Handle("PUT", "/users/{id:uint64}", updateUser)
  app.Listen(":8080")
}

func updateUser(ctx iris.Context) {
  id, _ := ctx.Params().GetUint64("id")

  var req request
  if err := ctx.ReadJSON(&req); err != nil {
    ctx.StopWithError(iris.StatusBadRequest, err)
    return
  }

  resp := response{
    ID:      id,
    Message: req.Firstname + " updated successfully",
  }
  ctx.JSON(resp)
}
```
> !برای اطلاعات بیشتر ، [مثال های مسیریابی](https://github.com/kataras/iris/blob/main/_examples/routing) را بخوانید

</details>

<details><summary>Handler with custom input and output arguments</summary>

[![https://github.com/kataras/iris/blob/main/_examples/dependency-injection/basic/main.go](https://user-images.githubusercontent.com/22900943/105253731-b8db6d00-5b88-11eb-90c1-0c92a5581c86.png)](https://twitter.com/iris_framework/status/1234783655408668672)

> اگر برایتان جالب بود [مثال های دیگری](https://github.com/kataras/iris/blob/main/_examples/dependency-injection) را مطالعه کنید

</details>

<details><summary>MVC</summary>

```go
package main

import (
  "github.com/kataras/iris/v12"
  "github.com/kataras/iris/v12/mvc"
)

type (
  request struct {
    Firstname string `json:"firstname"`
    Lastname  string `json:"lastname"`
  }

  response struct {
    ID      uint64 `json:"id"`
    Message string `json:"message"`
  }
)

func main() {
  app := iris.New()
  mvc.Configure(app.Party("/users"), configureMVC)
  app.Listen(":8080")
}

func configureMVC(app *mvc.Application) {
  app.Handle(new(userController))
}

type userController struct {
  // [...dependencies]
}

func (c *userController) PutBy(id uint64, req request) response {
  return response{
    ID:      id,
    Message: req.Firstname + " updated successfully",
  }
}
```
اگر به دنبال مثال‌های بیشتری هستید می‌توانید در [اینجا](_examples/mvc) مطالعه کنید
</details>
<div dir="rtl">
    
> دیگران درباره آیریس چه می گویند و برای پشتیبانی از پتانسیل‌های  این پروژه متن باز  می‌توانید از آن حمایت کنید

[![](https://iris-go.com/images/reviews.gif)](https://iris-go.com/testimonials/)

[![Benchmarks: Jul 18, 2020 at 10:46am (UTC)](https://iris-go.com/images/benchmarks.svg)](https://github.com/kataras/server-benchmarks)

## 👑 <a href="https://iris-go.com/donate">حامیان</a>
    
با کمک شما, ما می‌توانیم توسعه وب متن باز را برای همه بهبود ببخشیم !

> کمک هایی که تا حالا دریافت شده است !
 
## اموزش آیریس
    
### ساخت یک پروژه جدید

</div>
    
```sh
$ mkdir myapp
$ cd myapp
$ go mod init myapp
$ go get github.com/kataras/iris/v12@latest # or @v12.2.11
```

<div dir="rtl">
<summary>نصب بر روی پروژه موجود</summary>
</div>

```sh
$ cd myapp
$ go get github.com/kataras/iris/v12@latest
```

<div dir="rtl">
<summary>نصب با پرونده go.mod</summary>
</div>

```txt
module myapp

go 1.20

require github.com/kataras/iris/v12 v12.2.0-beta4.0.20220920072528-ff81f370625a
```
![](https://www.iris-go.com/images/gifs/install-create-iris.gif)

<div dir="rtl">
آیریس شامل مستندات گسترده و کاملی است که کار با چارچوب را آسان می کند.

> [مستندات](https://www.iris-go.com/docs)
    
برای اطلاعات بیشتر در مورد اسناد فنی می توانید به مستندات اصلی ما مراجعه کنید. 

> [مستندات اصلی](https://pkg.go.dev/github.com/kataras/iris/v12@main)
    
## دوست دارید در حین مسافرت کتاب بخوانید ?
    
 <a href="https://iris-go.com/#book"> <img alt="Book cover" src="https://iris-go.com/images/iris-book-cover-sm.jpg?v=12" /> </a>

[![follow author on twitter](https://img.shields.io/twitter/follow/makismaropoulos?color=3D8AA3&logoColor=3D8AA3&style=for-the-badge&logo=twitter)](https://twitter.com/intent/follow?screen_name=makismaropoulos)

[![follow Iris web framework on twitter](https://img.shields.io/twitter/follow/iris_framework?color=ee7506&logoColor=ee7506&style=for-the-badge&logo=twitter)](https://twitter.com/intent/follow?screen_name=iris_framework)

[![follow Iris web framework on facebook](https://img.shields.io/badge/Follow%20%40Iris.framework-522-2D88FF.svg?style=for-the-badge&logo=facebook)](https://www.facebook.com/iris.framework)
    
 امروز می توانید از طریق کتاب الکترونیکی آیریس (نسخه جدید ، آینده v12.2.0 +) دسترسی PDF و دسترسی آنلاین داشته باشید و در توسعه آیریس شرکت کنید.
    
 ## 🙌 مشارکت
    
 ما خیلی دوست داریم شما سهمی در توسعه چارچوب آیریس داشته باشید! برای دریافت اطلاعات بیشتر در مورد مشارکت در پروژه آیریس لطفاً پرونده [CONTRIBUTING.md](CONTRIBUTING.md) را مطالعه کنید.  
    
[لیست همه شرکت کنندگان](https://github.com/kataras/iris/graphs/contributors)
    
## 🛡 آسیب‌پذیری‌های امنیتی
    
اگر آسیب‌پذیری امنیتی در درون آیریس مشاهده کردید, لطفاً ایمیلی به [iris-go@outlook.com](mailto:iris-go@outlook.com) بفرستید. کلیه ضعف‌های امنیتی بلافاصله مورد توجه قرار خواهند گرفت.
    
## 📝 مجوز
    
این پروژه تحت پروانه [BSD 3-clause license](LICENSE) مجوز دارد ، دقیقاً مانند پروژه Go.    
    
نام پروژه "آیریس" از اساطیر یونانی الهام‌گرفته شده است.

</div>
  
