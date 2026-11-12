# <a href="https://iris-go.com"><img src="https://iris-go.com/images/logo-new-lq-45.png"></a> Iris Web Framework <a href="README_GR.md"><img width="20px" src="https://iris-go.com/images/flag-greece.svg" /> <a href="README_JA.md"><img width="20px" height="20px" src="https://iris-go.com/images/flag-japan.svg" /></a> </a> <a href="README_FR.md"><img width="20px" src="https://iris-go.com/images/flag-france.svg" /></a> <a href="README_ZH_HANT.md"><img width="20px" src="https://iris-go.com/images/flag-taiwan.svg" /></a> <a href="README_ZH_HANS.md"><img width="20px" src="https://iris-go.com/images/flag-china.svg" /></a> <a href="README_ES.md"><img width="20px" src="https://iris-go.com/images/flag-spain.png" /></a> <a href="README_FA.md"><img width="20px" src="https://iris-go.com/images/flag-iran.svg" /></a> <a href="README_RU.md"><img width="20px" src="https://iris-go.com/images/flag-russia.svg" /></a> <a href="README_KO.md"><img width="20px" src="https://iris-go.com/images/flag-south-korea.svg?v=12" /></a> <a href="README_PT_BR.md"><img width="20px" height="20px" src="https://iris-go.com/images/flag-brazil.svg" /></a> <a href="README_VN.md"><img width="20px" height="20px" src="https://iris-go.com/images/flag-vietnam.svg" /></a>

[![build status](https://img.shields.io/github/actions/workflow/status/kataras/iris/ci.yml?branch=main&style=for-the-badge)](https://github.com/kataras/iris/actions/workflows/ci.yml) [![release](https://img.shields.io/badge/release%20-v14.0-0077b3.svg?style=for-the-badge)](https://github.com/kataras/iris/releases) [![view examples](https://img.shields.io/badge/examples%20-59-a83adf.svg?style=for-the-badge&logo=go)](_examples/README.md) [![godocs](https://img.shields.io/badge/go-%20docs-488AC7.svg?style=for-the-badge)](https://pkg.go.dev/github.com/kataras/iris/v14@v14.0.0) [![chat](https://img.shields.io/gitter/room/iris_go/community.svg?color=cc2b5e&logo=gitter&style=for-the-badge)](https://gitter.im/iris_go/community) [![donate](https://img.shields.io/badge/support-Iris-blue.svg?style=for-the-badge&logo=paypal)](https://iris-go.com/donate)

Iris là một web framework nhanh, đầy đủ tính năng dành cho Go. Nó mang đến cho API hoặc website tiếp theo của bạn hệ thống định tuyến giàu biểu đạt với tham số đường dẫn có kiểu, một nơi duy nhất để xử lý lỗi, cơ chế dependency injection giúp mã nguồn luôn dễ kiểm thử, cùng các thiết lập mặc định an toàn, sẵn sàng cho production ngay từ đầu.

**Phiên bản 14** là bản phát hành lớn nhất trong lịch sử dự án. Framework đã được xây dựng lại xoay quanh generics của Go: một builder ứng dụng được trình biên dịch kiểm soát, các hàm hỗ trợ request có kiểu tự xác thực thay cho bạn, các controller được phân giải bởi container dependency injection, và một pipeline xử lý lỗi chỉ cần cấu hình một lần. [HISTORY.md](HISTORY.md) ghi lại mọi điểm mới; [Hướng dẫn nâng cấp](MIGRATION.md) đưa ứng dụng v12 lên phiên bản mới chỉ trong một lần ngồi.

> 🤖 **Hỏi AI về Iris.** [Code Wiki cho Iris](https://codewiki.google/github.com/kataras/iris): miễn phí, do Google tạo trực tiếp từ repository này và luôn đồng bộ với nó theo từng commit. Bạn có thể duyệt kiến trúc qua các sơ đồ, hoặc đặt câu hỏi bằng ngôn ngữ thường ngày cho phần chat của nó về codebase và nhận câu trả lời dựa trên chính mã nguồn thực tế.

Xem [những gì người khác nói về Iris](https://www.iris-go.com/#review) và **[thả sao](https://github.com/kataras/iris/stargazers)** cho dự án mã nguồn mở này để ủng hộ sự phát triển của nó. Tò mò về tốc độ? Hãy xem [các bài benchmark máy chủ](https://github.com/kataras/server-benchmarks) độc lập.

[![](https://iris-go.com/images/reviews.gif)](https://iris-go.com/testimonials/)

```go
package main

import "github.com/kataras/iris/v14"

func main() {
    app := iris.New()

    app.Get("/", func(ctx iris.Context) {
        ctx.JSON(iris.Map{"message": "Hello, World!"})
    })

    app.Get("/greet/{name:string}", func(ctx iris.Context) {
        ctx.WriteString("Hello, " + ctx.Params().GetString("name"))
    })

    app.Listen(":8080")
}
```

<details><summary>Cả stack production, trong một biểu thức duy nhất</summary>

Các ứng dụng thực tế dùng `iris.NewBuilder`: một chuỗi gọi được trình biên dịch kiểm soát, kết nối CORS, nén dữ liệu, log truy cập, endpoint health, bảng ánh xạ lỗi, các service và các nhóm API của bạn. Sai thứ tự các bước là trình biên dịch từ chối ngay.

```go
iris.NewBuilder().
    Prefix("/api").
    AllowOrigin("*").
    Compression(true).
    LogRequests(true).
    Health(true, "production", "kataras").
    Errors(errors.NewOptions().
        MapErrors(errors.NotFound, catalog.ErrNotFound)).
    Services(catalog.NewRepository, catalog.NewService).
    RouterMiddlewares().
    Middlewares().
    API("/products", api.NewProductsAPI).
    Build().
    Listen(":8080")
```

Handler chỉ dài khoảng sáu dòng, bởi việc giải mã, xác thực và kết xuất lỗi đều diễn ra ở một nơi đã được cấu hình sẵn:

```go
func (api *ProductsAPI) create(ctx iris.Context) {
    input, ok := ctx.BindJSON[catalog.ProductInput]()
    if !ok {
        return // 400 already written: malformed JSON or failed Validate.
    }

    id, err := api.svc.Create(ctx, input)
    ctx.Created(id, err) // 201, or the centrally mapped error.
}
```

[Hướng dẫn bắt đầu](TUTORIAL.md) xây dựng một service hoàn chỉnh, có kiểm thử, trên chính stack này chỉ trong một tài liệu.

</details>


Như một [lập trình viên Go](https://twitter.com/dkuye/status/1532087942696554497) từng nói, **Iris lo cho bạn mọi mặt và vẫn đứng vững suốt bao năm qua**.

Những gì bạn có với Iris v14:

* Định tuyến với tham số đường dẫn có kiểu (`{id:uuid}`, `{n:int}`, `{addr:email}` và nhiều hơn nữa), macro tùy biến và các ràng buộc. Dữ liệu không hợp lệ bị từ chối trước khi handler của bạn chạy.
* `iris.NewBuilder`: CORS, nén, log truy cập, endpoint health, ánh xạ lỗi, service và các nhóm API được lắp ráp trong một chuỗi gọi được kiểm tra lúc biên dịch.
* Dependency injection: constructor khai báo thứ mình cần, container sẽ cung cấp. Hook `Initializer` lúc khởi động, tự động gọi các closer khi tắt.
* Bộ helper `rest` và một bảng ánh xạ lỗi tập trung: handler mỏng gọn, lỗi chuẩn hóa mà máy đọc được, chi tiết nội bộ không bao giờ lộ ra phía client.
* Payload request tự xác thực: chỉ cần triển khai `Validate() error`, `ctx.BindJSON` sẽ chạy nó ngay sau khi giải mã.
* Ba mươi mốt middleware tích hợp sẵn: accesslog, API key, basicauth, bodylimit, cache, CORS, hCaptcha và reCAPTCHA, JWT, monitor, pprof, rate limiting, recovery, request ID, rewrite, session, versioning và nhiều hơn nữa.
* Xác thực ở mọi quy mô: Basic Auth chỉ một dòng, API key, SDK JWT dùng generics, một identity server hoàn chỉnh và một OAuth2 provider.
* Controller MVC xây dựng trên generics của Go.
* WebSocket, phục vụ gRPC và versioning cho API.
* View engine: HTML, Django, Handlebars, Pug và nhiều hơn nữa thông qua iris-contrib/views.
* Bản địa hóa (i18n) với hỗ trợ số nhiều và nhận diện ngôn ngữ, kèm sitemap.
* Session với các mặc định cookie an toàn (`Secure`, `SameSite=Lax` ngay từ đầu).
* Kiểu phản hồi phong phú: JSON, JSONP, XML, YAML, Markdown, Protocol Buffers, MessagePack, content negotiation, streaming và Server-Sent Events.
* Backend AI: package `ai` kết xuất và chuyển tiếp đầu ra streaming của mô hình qua HTTP (text delta, reasoning trace, các mảnh tool call, usage). Không phụ thuộc vào nhà cung cấp nào và không kèm theo SDK của hãng nào.
* Một HTTP client, package `client`, để gọi các service khác theo cùng bộ quy tắc JSON mà phía server sử dụng.
* Nén phản hồi: gzip, deflate, brotli, snappy, s2, zstd.
* TLS với Let's Encrypt, cùng tunnel ngrok tích hợp sẵn (opt-in: `import _ "github.com/kataras/iris/v14/core/server/ngrok"`) để đưa ứng dụng đang phát triển lên internet trong vài giây.
* Kiểm thử HTTP trong bộ nhớ: test toàn bộ ứng dụng không cần cổng, không cần mạng, không cần sleep.
* Một cuốn sách hoàn chỉnh có thể đọc trực tuyến và tải về miễn phí, một tutorial thực hành và một AI wiki trực tuyến (xem mục Học Iris bên dưới).
* Và quan trọng nhất: bạn luôn nhận được câu trả lời và sự hỗ trợ nhanh chóng, từ ngày đầu tiên cho đến tận bây giờ.

## 📖 Học Iris

> Mới làm quen với Iris? Hãy bắt đầu với [Hướng dẫn bắt đầu](TUTORIAL.md): một tài liệu duy nhất đưa bạn từ `go mod init` đến một REST API đã được kiểm thử, mang dáng dấp production. Đang nâng cấp từ v12? Xem [Hướng dẫn nâng cấp](MIGRATION.md). Có thắc mắc ở bất kỳ bước nào? Hãy hỏi **AI wiki trực tuyến** miễn phí tại [codewiki.google/github.com/kataras/iris](https://codewiki.google/github.com/kataras/iris): do Google tạo từ mã nguồn Iris và luôn được đồng bộ với nó, kèm phần chat tích hợp trả lời các câu hỏi về codebase.

### Cài đặt

Yêu cầu duy nhất là [ngôn ngữ lập trình Go](https://go.dev/dl/), phiên bản 1.27 trở lên.

#### Tạo dự án mới

```sh
$ mkdir myapp
$ cd myapp
$ go mod init myapp
$ go get github.com/kataras/iris/v14@latest # or @v14.0.0
```

<details><summary>Cài đặt vào dự án có sẵn</summary>

```sh
$ cd myapp
$ go get github.com/kataras/iris/v14@latest
```

**Chạy**

```sh
$ go mod tidy
$ go run .
```

</details>

![](https://www.iris-go.com/images/gifs/install-create-iris.gif)

Iris có bộ **[tài liệu](https://www.iris-go.com/docs)** phong phú và kỹ lưỡng, giúp bạn dễ dàng bắt đầu với framework.

Để xem tài liệu kỹ thuật chi tiết hơn, hãy ghé [godocs](https://pkg.go.dev/github.com/kataras/iris/v14@v14.0.0) của chúng tôi. Để có phần giới thiệu trọn vẹn, thực hành về stack v14 được khuyến nghị, hãy đọc [Hướng dẫn bắt đầu](TUTORIAL.md); còn nếu bạn đang nâng cấp từ v12, hãy làm theo [Hướng dẫn nâng cấp](MIGRATION.md).

Bạn thích học từ mã nguồn? [**59 ví dụ**](_examples/README.md) đi từ route đầu tiên đến dependency injection, xác thực, kiểm thử và các điểm mở rộng của framework. Mỗi ví dụ đều đã được biên dịch và chạy thử qua HTTP trước khi commit.

### Sách Iris

<a href="https://iris-go.com/book/"> <img alt="Book cover" src="https://iris-go.com/images/iris-book-cover-sm.jpg?v=14" /> </a>

Cuốn sách hoàn chỉnh gồm một lời mở đầu, 24 chương và một lời kết, mọi ví dụ mã đều được đối chiếu với mã nguồn của framework. Mỗi chương mở đầu bằng phần bối cảnh dễ hiểu dành cho người mới và khép lại bằng danh mục đọc thêm chọn lọc. Mọi chương đều đọc miễn phí tại [iris-go.com/docs](https://iris-go.com/docs/); [trang sách](https://iris-go.com/book/) có mục lục đầy đủ.

Bản PDF cũng miễn phí. [Tải trực tiếp](https://iris-go.com/iris-book.pdf): không có biểu mẫu nào phải điền và không cần yêu cầu gì cả. Nếu cuốn sách giúp ích cho bạn, [tài trợ cho dự án](https://iris-go.com/donate/) là cách để bạn góp phần vào sự phát triển của Iris.

[![follow author on twitter](https://img.shields.io/twitter/follow/makismaropoulos?color=3D8AA3&logoColor=3D8AA3&style=for-the-badge&logo=twitter)](https://twitter.com/intent/follow?screen_name=makismaropoulos)

[![follow Iris web framework on twitter](https://img.shields.io/twitter/follow/iris_framework?color=ee7506&logoColor=ee7506&style=for-the-badge&logo=twitter)](https://twitter.com/intent/follow?screen_name=iris_framework)

[![follow Iris web framework on facebook](https://img.shields.io/badge/Follow%20%40Iris.framework-569-2D88FF.svg?style=for-the-badge&logo=facebook)](https://www.facebook.com/iris.framework)

## 🤖 Công cụ AI cho Iris

Có hai công cụ AI hiểu riêng framework này, vượt ra ngoài những gì một mô hình tổng quát biết về Go.

[AI wiki trực tiếp](https://codewiki.google/github.com/kataras/iris) miễn phí và chạy ngay trên trình duyệt: xem kiến trúc hoặc hỏi chat của nó về mã nguồn.

[Plexon AI](https://plexon.ai) là một trợ lý desktop cho Windows, macOS và Linux, do tác giả của Iris xây dựng. Nó tích hợp sẵn skill Iris ([iris-go.com/ai](https://iris-go.com/ai/)): API v14, chuỗi builder, các helper `rest`, ánh xạ lỗi, SDK xác thực và các mẫu kiểm thử, trong hai mươi mốt tài liệu tham khảo mà trợ lý nạp khi cần trong lúc viết mã cho bạn. Bật skill trong bảng Skills, hoặc cài [persona Software Developer](https://plexon.ai/personas/software-developer/), persona này sẽ bật skill cùng với các công cụ phát triển còn lại. [Tải Plexon AI](https://plexon.ai/download/).

---

Về những người ủng hộ, cách đóng góp, chính sách bảo mật và giấy phép, xem [README tiếng Anh](README.md).

Dự án này được cấp phép theo [giấy phép BSD 3-clause](LICENSE), giống như chính dự án Go.
