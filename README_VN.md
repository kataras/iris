<!--<h1><img width="24" height="25" src ="https://www.iris-go.com/images/logo-new-lq-45.png"/> News</h1>

 Iris version **12.2.0** has been [released](HISTORY.md#sa-11-march-2023--v1220)! As always, the latest version of Iris comes with the promise of lifetime active maintenance.

Try the official [Iris Command Line Interface](https://github.com/kataras/iris-cli) today! -->

# <a href="https://iris-go.com"><img src="https://iris-go.com/images/logo-new-lq-45.png"></a> Iris Web Framework <a href="README_GR.md"><img width="20px" src="https://iris-go.com/images/flag-greece.svg" /></a> <a href="README_FR.md"><img width="20px" src="https://iris-go.com/images/flag-france.svg" /></a> <a href="README_ZH_HANT.md"><img width="20px" src="https://iris-go.com/images/flag-taiwan.svg" /></a> <a href="README_ZH_HANS.md"><img width="20px" src="https://iris-go.com/images/flag-china.svg" /></a> <a href="README_ES.md"><img width="20px" src="https://iris-go.com/images/flag-spain.png" /></a> <a href="README_FA.md"><img width="20px" src="https://iris-go.com/images/flag-iran.svg" /></a> <a href="README_RU.md"><img width="20px" src="https://iris-go.com/images/flag-russia.svg" /></a> <a href="README_KO.md"><img width="20px" src="https://iris-go.com/images/flag-south-korea.svg?v=12" /></a> <a href="README_PT_BR.md"><img width="20px" height="20px" src="https://iris-go.com/images/flag-brazil.svg" /></a>

[![build status](https://img.shields.io/github/actions/workflow/status/kataras/iris/ci.yml?branch=main&style=for-the-badge)](https://github.com/kataras/iris/actions/workflows/ci.yml) [![view examples](https://img.shields.io/badge/examples%20-285-a83adf.svg?style=for-the-badge&logo=go)](https://github.com/kataras/iris/tree/main/_examples) [![chat](https://img.shields.io/gitter/room/iris_go/community.svg?color=cc2b5e&logo=gitter&style=for-the-badge)](https://gitter.im/iris_go/community) <!--[![FOSSA Status](https://img.shields.io/badge/LICENSE%20SCAN-PASSING❤️-CD2956?style=for-the-badge&logo=fossa)](https://app.fossa.io/projects/git%2Bgithub.com%2Fkataras%2Firis?ref=badge_shield)--> [![donate](https://img.shields.io/badge/support-Iris-blue.svg?style=for-the-badge&logo=paypal)](https://iris-go.com/donate) <!--[![report card](https://img.shields.io/badge/report%20card-a%2B-ff3333.svg?style=for-the-badge)](https://goreportcard.com/report/github.com/kataras/iris)--><!--[![godocs](https://img.shields.io/badge/go-%20docs-488AC7.svg?style=for-the-badge)](https://pkg.go.dev/github.com/kataras/iris/v12@v12.2.10)--> <!-- [![release](https://img.shields.io/badge/release%20-v12.0-0077b3.svg?style=for-the-badge)](https://github.com/kataras/iris/releases) -->

Iris là một khung web nhanh, đơn giản nhưng đầy đủ tính năng và rất hiệu quả dành cho Go.

Nó cung cấp một nền tảng đẹp mắt và dễ sử dụng cho trang web hoặc API tiếp theo của bạn.


Tìm hiểu xem [những người khác nói gì về Iris](https://www.iris-go.com/#review) và **[gắn sao](https://github.com/kataras/iris/stargazers)** dự án mã nguồn mở này để phát huy tiềm năng của nó.

[![](https://iris-go.com/images/reviews.gif)](https://iris-go.com/testimonials/)

[![Benchmarks: Jul 18, 2020 at 10:46am (UTC)](https://iris-go.com/images/benchmarks.svg)](https://github.com/kataras/server-benchmarks)

```go
package main

import "github.com/kataras/iris/v12"

func main() {
  app := iris.New()
  app.Use(iris.Compression)

  app.Get("/", func(ctx iris.Context) {
    ctx.HTML("Xin chào <strong>%s</strong>!", "Thế Giới")
  })

  app.Listen(":8080")
}
```

<!-- <details><summary>More with simple Handler</summary>

```go
package main

import "github.com/kataras/iris/v12"

type (
  request struct {
    Firstname string `json:"firstname"`
    Lastname  string `json:"lastname"`
  }

  response struct {
    ID      string `json:"id"`
    Message string `json:"message"`
  }
)

func main() {
  app := iris.New()
  app.Handle("PUT", "/users/{id:uuid}", updateUser)
  app.Listen(":8080")
}

func updateUser(ctx iris.Context) {
  id := ctx.Params().Get("id")

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

> Read the [routing examples](https://github.com/kataras/iris/blob/main/_examples/routing) for more!

</details>

<details><summary>Handler with custom input and output arguments</summary>

[![https://github.com/kataras/iris/blob/main/_examples/dependency-injection/basic/main.go](https://user-images.githubusercontent.com/22900943/105253731-b8db6d00-5b88-11eb-90c1-0c92a5581c86.png)](https://twitter.com/iris_framework/status/1234783655408668672)

> Interesting? Read the [examples](https://github.com/kataras/iris/blob/main/_examples/dependency-injection).

</details>

<details><summary>Party Controller (NEW)</summary>

> Head over to the [full running example](https://github.com/kataras/iris/blob/main/_examples/routing/party-controller)!

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

Want to see more? Navigate through [mvc examples](_examples/mvc)!
</details>


<details><summary>API Guide <strong>HOT</strong></summary>

```go
package main

import (
  // [other packages...]

  "github.com/kataras/iris/v12"
)

func main() {
  iris.NewGuide().
    AllowOrigin("*").
    Compression(true).
    Health(true, "development", "kataras").
    Timeout(0, 20*time.Second, 20*time.Second).
    Middlewares(basicauth.New(...)).
    Services(
        // NewDatabase(),
        // NewPostgresRepositoryRegistry,
        // NewUserService,
    ).
    API("/users", new(UsersAPI)).
    Listen(":80")
}
```

</details>

<br/>

-->

Như một [nhà phát triển Go](https://twitter.com/dkuye/status/1532087942696554497) đã từng nói, **Iris giúp bạn bảo vệ toàn diện và đứng vững qua nhiều năm**.

Một số tính năng Iris cung cấp:

* HTTP/2 (Push, cả những Embedded data)
* Middleware (Accesslog, Basicauth, CORS, gRPC, Anti-Bot hCaptcha, JWT, MethodOverride, ModRevision, Monitor, PPROF, Ratelimit, Anti-Bot reCaptcha, Recovery, RequestID, Rewrite)
* API Versioning
* Model-View-Controller
* Websockets
* gRPC
* Auto-HTTPS
* Tích hợp hỗ trợ ngrok để đưa ứng dụng của bạn lên internet một cách nhanh nhất
* Unique Router với đường dẫn động làm tham số với các loại tiêu chuẩn như :uuid, :string, :int... và khả năng tạo của riêng bạn
* Compression
* View Engines (HTML, Django, Handlebars, Pug/Jade and more)
* Tạo Máy chủ tệp của riêng bạn và lưu trữ máy chủ WebDAV của riêng bạn
* Cache
* Localization (i18n, sitemap)
* Sessions
* Rich Responses (HTML, Text, Markdown, XML, YAML, Binary, JSON, JSONP, Protocol Buffers, MessagePack, Content Negotiation, Streaming, Server-Sent Events and more)
* Response Compression (gzip, deflate, brotli, snappy, s2)
* Rich Requests (Bind URL Query, Headers, Form, Text, XML, YAML, Binary, JSON, Validation, Protocol Buffers, MessagePack and more)
* Dependency Injection (MVC, Handlers, API Routers)
* Testing Suite
* Và điều quan trọng nhất... bạn nhận được câu trả lời và hỗ trợ nhanh chóng từ ngày đầu tiên cho đến bây giờ - đó là sáu năm đầy đủ!

## 👑 <a href="https://iris-go.com/donate">Người ủng hộ</a>

Với sự giúp đỡ của bạn, chúng tôi có thể cải thiện việc phát triển web Nguồn mở cho mọi người!

## 📖 Học Iris

### Cài đặt

Yêu cầu duy nhất là [Ngôn ngữ lập trình Go](https://go.dev/dl/).

#### Tạo một dự án mới

```sh
$ mkdir myapp
$ cd myapp
$ go mod init myapp
$ go get github.com/kataras/iris/v12@latest # or @v12.2.10
```

<details><summary>Cài đặt trên dự án hiện có</summary>

```sh
$ cd myapp
$ go get github.com/kataras/iris/v12@latest
```

**Run**

```sh
$ go mod tidy -compat=1.20 # -compat="1.20" for windows.
$ go run .
```

</details>

![](https://www.iris-go.com/images/gifs/install-create-iris.gif)

Iris chứa **[tài liệu](https://www.iris-go.com/docs)** phong phú và kỹ lưỡng giúp bạn dễ dàng bắt đầu với khung.

<!-- Iris contains extensive and thorough **[wiki](https://github.com/kataras/iris/wiki)** making it easy to get started with the framework. -->

<!-- ![](https://media.giphy.com/media/Ur8iqy9FQfmPuyQpgy/giphy.gif) -->

Để có tài liệu kỹ thuật chi tiết hơn, bạn có thể truy cập [godocs](https://pkg.go.dev/github.com/kataras/iris/v12@main) của chúng tôi. Và đối với mã thực thi, bạn luôn có thể truy cập thư mục con của kho lưu trữ [./_examples](_examples).

### Bạn có thích đọc khi đi du lịch không?

<a href="https://iris-go.com/#book"> <img alt="Book cover" src="https://iris-go.com/images/iris-book-cover-sm.jpg?v=12" /> </a>

[![follow author on twitter](https://img.shields.io/twitter/follow/makismaropoulos?color=3D8AA3&logoColor=3D8AA3&style=for-the-badge&logo=twitter)](https://twitter.com/intent/follow?screen_name=makismaropoulos)

[![follow Iris web framework on twitter](https://img.shields.io/twitter/follow/iris_framework?color=ee7506&logoColor=ee7506&style=for-the-badge&logo=twitter)](https://twitter.com/intent/follow?screen_name=iris_framework)

[![follow Iris web framework on facebook](https://img.shields.io/badge/Follow%20%40Iris.framework-569-2D88FF.svg?style=for-the-badge&logo=facebook)](https://www.facebook.com/iris.framework)

Bạn có thể [yêu cầu](https://www.iris-go.com/#ebookDonateForm) PDF và truy cập trực tuyến **Sách điện tử Iris** (Phiên bản mới, **tương lai v12.2.0+**) hôm nay và được tham gia vào sự phát triển của Iris.

## 🙌 Đóng góp

Chúng tôi muốn thấy sự đóng góp của bạn cho Iris Web Framework! Để biết thêm thông tin về việc đóng góp cho dự án Iris, vui lòng kiểm tra tệp [CONTRIBUTING.md](CONTRIBUTING.md).

[Danh sách những người đóng góp](https://github.com/kataras/iris/graphs/contributors)

## 🛡 Lỗ hổng bảo mật

Nếu bạn phát hiện ra lỗ hổng bảo mật trong Iris, vui lòng gửi e-mail tới [iris-go@outlook.com](mailto:iris-go@outlook.com). Tất cả các lỗ hổng bảo mật sẽ được giải quyết kịp thời.

## 📝 Giấy phép

Dự án này được cấp phép theo [BSD 3-clause license](LICENSE), giống như chính dự án Go.

Tên dự án "Iris" được lấy cảm hứng từ thần thoại Hy Lạp.
<!-- ## Stargazers over time

[![Stargazers over time](https://starchart.cc/kataras/iris.svg)](https://starchart.cc/kataras/iris) -->
