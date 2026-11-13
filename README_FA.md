# <a href="https://iris-go.com"><img src="https://iris-go.com/images/logo-new-lq-45.png"></a> Iris Web Framework <a href="README_GR.md"><img width="20px" src="https://iris-go.com/images/flag-greece.svg" /> <a href="README_JA.md"><img width="20px" height="20px" src="https://iris-go.com/images/flag-japan.svg" /></a> </a> <a href="README_FR.md"><img width="20px" src="https://iris-go.com/images/flag-france.svg" /></a> <a href="README_ZH_HANT.md"><img width="20px" src="https://iris-go.com/images/flag-taiwan.svg" /></a> <a href="README_ZH_HANS.md"><img width="20px" src="https://iris-go.com/images/flag-china.svg" /></a> <a href="README_ES.md"><img width="20px" src="https://iris-go.com/images/flag-spain.png" /></a> <a href="README_FA.md"><img width="20px" src="https://iris-go.com/images/flag-iran.svg" /></a> <a href="README_RU.md"><img width="20px" src="https://iris-go.com/images/flag-russia.svg" /></a> <a href="README_KO.md"><img width="20px" src="https://iris-go.com/images/flag-south-korea.svg?v=12" /></a> <a href="README_PT_BR.md"><img width="20px" height="20px" src="https://iris-go.com/images/flag-brazil.svg" /></a> <a href="README_VN.md"><img width="20px" height="20px" src="https://iris-go.com/images/flag-vietnam.svg" /></a>

[![build status](https://img.shields.io/github/actions/workflow/status/kataras/iris/ci.yml?branch=main&style=for-the-badge)](https://github.com/kataras/iris/actions/workflows/ci.yml) [![release](https://img.shields.io/badge/release%20-v14.0-0077b3.svg?style=for-the-badge)](https://github.com/kataras/iris/releases) [![view examples](https://img.shields.io/badge/examples%20-59-a83adf.svg?style=for-the-badge&logo=go)](_examples/README.md) [![godocs](https://img.shields.io/badge/go-%20docs-488AC7.svg?style=for-the-badge)](https://pkg.go.dev/github.com/kataras/iris/v14@v14.0.1) [![chat](https://img.shields.io/gitter/room/iris_go/community.svg?color=cc2b5e&logo=gitter&style=for-the-badge)](https://gitter.im/iris_go/community) [![donate](https://img.shields.io/badge/support-Iris-blue.svg?style=for-the-badge&logo=paypal)](https://iris-go.com/donate)

Iris یک فریم‌ورک وب سریع و کامل (batteries-included) برای زبان Go است. این فریم‌ورک برای API یا وب‌سایت بعدی شما مسیریابی گویا با پارامترهای مسیرِ دارای نوع، یک نقطهٔ واحد برای مدیریت خطاها، تزریق وابستگی که کد را تست‌پذیر نگه می‌دارد و پیش‌فرض‌های production که از همان ابتدا امن هستند فراهم می‌کند.

**نسخهٔ 14** بزرگ‌ترین انتشار در تاریخ این پروژه است. فریم‌ورک بر پایهٔ جنریک‌های Go بازسازی شده است: یک سازندهٔ برنامه که در زمان کامپایل اعمال می‌شود، هِلپرهای درخواستِ دارای نوع که اعتبارسنجی را برای شما انجام می‌دهند، کنترلرهایی که توسط کانتینر وابستگی حل می‌شوند و یک خط لولهٔ خطا که فقط یک بار پیکربندی می‌کنید. [HISTORY.md](HISTORY.md) همهٔ موارد جدید را ثبت می‌کند؛ [راهنمای مهاجرت](MIGRATION.md) برنامه‌های v12 را در یک نشست به نسخهٔ جدید می‌رساند.

> 🤖 **دربارهٔ Iris از هوش مصنوعی بپرسید.** [Code Wiki رایگان Iris](https://codewiki.google/github.com/kataras/iris) که Google آن را از همین مخزن تولید می‌کند، کامیت به کامیت با آن همگام می‌ماند. معماری را با نمودارها مرور کنید، یا از چت آن به زبان ساده دربارهٔ کدبیس سؤال بپرسید و پاسخ‌هایی مبتنی بر سورس واقعی دریافت کنید.

ببینید [دیگران دربارهٔ Iris چه می‌گویند](https://www.iris-go.com/#review) و برای حمایت از توسعهٔ این پروژهٔ متن‌باز به آن **[ستاره](https://github.com/kataras/iris/stargazers)** بدهید. دربارهٔ سرعت کنجکاوید؟ [بنچمارک‌های مستقل سرور](https://github.com/kataras/server-benchmarks) را ببینید.

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

<details><summary>استک production، در یک عبارت</summary>

برنامه‌های واقعی از `iris.NewBuilder` استفاده می‌کنند: زنجیره‌ای که در زمان
کامپایل بررسی می‌شود و CORS، فشرده‌سازی، لاگ‌های دسترسی، یک endpoint سلامت،
نگاشت خطاهای شما، سرویس‌ها و گروه‌های API شما را به هم متصل می‌کند. کامپایلر
ترتیب نادرست مراحل را رد می‌کند.

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

هندلرها شش‌خطی می‌مانند، چون رمزگشایی، اعتبارسنجی و رندر خطا در یک نقطهٔ
پیکربندی‌شده انجام می‌شوند:

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

[آموزش شروع به کار](TUTORIAL.md) در یک سند، سرویسی کامل و تست‌شده روی همین
استک می‌سازد.

</details>


همان‌طور که یک [توسعه‌دهندهٔ Go](https://twitter.com/dkuye/status/1532087942696554497) زمانی گفت، **Iris از هر جهت هوای شما را دارد و سال‌هاست استوار ایستاده است**.

آنچه با Iris v14 به دست می‌آورید:

* مسیریابی با پارامترهای مسیرِ دارای نوع (`{id:uuid}`، `{n:int}`، `{addr:email}` و بیشتر)، ماکروها و محدودیت‌های سفارشی. ورودی نامعتبر پیش از اجرای هندلر شما رد می‌شود.
* `iris.NewBuilder`: ‏CORS، فشرده‌سازی، لاگ دسترسی، endpoint سلامت، نگاشت خطا، سرویس‌ها و گروه‌های API، همه در یک زنجیرهٔ بررسی‌شده در زمان کامپایل.
* تزریق وابستگی: سازنده‌ها اعلام می‌کنند چه چیزی نیاز دارند و کانتینر آن را فراهم می‌کند. هوک‌های `Initializer` هنگام راه‌اندازی، و بسته‌شدن خودکار منابع هنگام خاموش‌شدن.
* هلپرهای `rest` و یک نگاشت مرکزی خطا: هندلرهای نازک، خطاهای استاندارد و ماشین‌خوان، و جزئیات داخلی که هرگز به کلاینت‌ها نشت نمی‌کنند.
* بدنه‌های درخواست که خودشان را اعتبارسنجی می‌کنند: `Validate() error` را پیاده‌سازی کنید تا `ctx.BindJSON` آن را پس از رمزگشایی اجرا کند.
* سی و یک middleware داخلی: accesslog، کلیدهای API، basicauth، bodylimit، cache، CORS، hCaptcha و reCAPTCHA، JWT، monitor، pprof، محدودسازی نرخ درخواست، recovery، شناسهٔ درخواست، rewrite، سشن‌ها، نسخه‌بندی و بیشتر.
* احراز هویت در هر مقیاس: Basic Auth یک‌خطی، کلیدهای API، یک JWT SDK جنریک، یک سرور هویت کامل و یک ارائه‌دهندهٔ OAuth2.
* کنترلرهای MVC ساخته‌شده بر پایهٔ جنریک‌های Go.
* WebSocket، سرو کردن gRPC و نسخه‌بندی API.
* موتورهای view: ‏HTML، Django، Handlebars، Pug و بیشتر از طریق iris-contrib/views.
* بومی‌سازی با پشتیبانی از حالت جمع و تشخیص زبان، به‌همراه sitemap.
* سشن‌ها با پیش‌فرض‌های امن کوکی (`Secure` و `SameSite=Lax` به‌صورت پیش‌فرض).
* پاسخ‌های غنی: JSON، JSONP، XML، YAML، Markdown، Protocol Buffers، MessagePack، مذاکرهٔ محتوا، استریمینگ و Server-Sent Events.
* بک‌اندهای هوش مصنوعی: پکیج `ai` خروجی استریم‌شدهٔ مدل را روی HTTP رندر و رله می‌کند (دلتاهای متنی، ردپای استدلال، تکه‌های فراخوانی ابزار، میزان مصرف). به هیچ ارائه‌دهنده‌ای وابسته نیست و SDK هیچ‌کدام را هم همراه ندارد.
* یک کلاینت HTTP، یعنی پکیج `client`، برای فراخوانی سرویس‌های دیگر با همان قواعد JSON که سمت سرور به کار می‌برد.
* فشرده‌سازی پاسخ: gzip، deflate، brotli، snappy، s2، zstd.
* TLS با Let's Encrypt، و تونل داخلی ngrok (opt-in: `import _ "github.com/kataras/iris/v14/core/server/ngrok"`) برای آنلاین‌کردن برنامهٔ در حال توسعه در چند ثانیه.
* تست HTTP درون‌حافظه‌ای: تست کل برنامه بدون پورت، بدون شبکه و بدون sleep.
* یک کتاب کامل که خواندن آنلاین و دانلود آن رایگان است، یک آموزش عملی و یک ویکی هوش مصنوعی زنده (بخش «یادگیری Iris» را در ادامه ببینید).
* و مهم‌تر از همه: پاسخ‌های سریع و پشتیبانی، از روز نخست تا امروز.

## 📖 یادگیری Iris

> با Iris تازه آشنا شده‌اید؟ از [آموزش شروع به کار](TUTORIAL.md) شروع کنید:
> یک سند که شما را از `go mod init` تا یک REST API تست‌شده با ساختار
> production می‌رساند. از v12 ارتقا می‌دهید؟ [راهنمای مهاجرت](MIGRATION.md)
> را ببینید. در هر مرحله سؤالی داشتید؟ از **ویکی هوش مصنوعی زندهٔ** رایگان در
> [codewiki.google/github.com/kataras/iris](https://codewiki.google/github.com/kataras/iris)
> بپرسید. Google آن را از سورس Iris تولید و با آن همگام نگه می‌دارد و چت
> داخلی‌اش به سؤال‌های شما دربارهٔ کدبیس پاسخ می‌دهد.

### نصب

تنها پیش‌نیاز، [زبان برنامه‌نویسی Go](https://go.dev/dl/) نسخهٔ 1.27 یا بالاتر است.

#### ساخت پروژهٔ جدید

```sh
$ mkdir myapp
$ cd myapp
$ go mod init myapp
$ go get github.com/kataras/iris/v14@latest # or @v14.0.1
```

<details><summary>نصب روی پروژهٔ موجود</summary>

```sh
$ cd myapp
$ go get github.com/kataras/iris/v14@latest
```

**اجرا**

```sh
$ go mod tidy
$ go run .
```

</details>

![](https://www.iris-go.com/images/gifs/install-create-iris.gif)

Iris دارای **[مستندات](https://www.iris-go.com/docs)** گسترده و دقیقی است که شروع کار با فریم‌ورک را آسان می‌کند.

برای مستندات فنی جزئی‌تر به [godocs](https://pkg.go.dev/github.com/kataras/iris/v14@v14.0.1) ما سر بزنید. برای آشنایی کامل و عملی با استک پیشنهادی v14، [آموزش شروع به کار](TUTORIAL.md) را بخوانید و اگر از v12 ارتقا می‌دهید، [راهنمای مهاجرت](MIGRATION.md) را دنبال کنید.

ترجیح می‌دهید از روی کد یاد بگیرید؟ [**59 نمونه**](_examples/README.md) از اولین مسیر شروع می‌شوند و تا تزریق وابستگی، احراز هویت، تست و نقاط توسعه‌پذیری فریم‌ورک پیش می‌روند. هر کدام پیش از commit شدن کامپایل شده و روی HTTP آزموده شده‌اند.

### کتاب Iris

<a href="https://iris-go.com/book/"> <img alt="Book cover" src="https://iris-go.com/images/iris-book-cover-sm.jpg?v=14" /> </a>

کتاب کامل شامل یک پیش‌گفتار، 24 فصل و یک مؤخره؛ هر نمونه کد با سورس فریم‌ورک تطبیق داده و راستی‌آزمایی شده است. هر فصل با پیش‌زمینه‌ای به زبان ساده برای تازه‌کارها آغاز می‌شود و با منابع منتخب برای مطالعهٔ بیشتر پایان می‌یابد. همهٔ فصل‌ها به‌صورت رایگان در [iris-go.com/docs](https://iris-go.com/docs/) قابل خواندن است و فهرست کامل در [صفحهٔ کتاب](https://iris-go.com/book/) آمده است.

نسخهٔ PDF هم رایگان است. [آن را مستقیم دانلود کنید](https://iris-go.com/iris-book.pdf): نیازی به پر کردن فرم یا ثبت درخواست نیست. اگر کتاب برایتان مفید بود، [حمایت مالی از پروژه](https://iris-go.com/donate/) راه سهیم شدن در توسعهٔ Iris است.

[![follow author on twitter](https://img.shields.io/twitter/follow/makismaropoulos?color=3D8AA3&logoColor=3D8AA3&style=for-the-badge&logo=twitter)](https://twitter.com/intent/follow?screen_name=makismaropoulos)

[![follow Iris web framework on twitter](https://img.shields.io/twitter/follow/iris_framework?color=ee7506&logoColor=ee7506&style=for-the-badge&logo=twitter)](https://twitter.com/intent/follow?screen_name=iris_framework)

[![follow Iris web framework on facebook](https://img.shields.io/badge/Follow%20%40Iris.framework-569-2D88FF.svg?style=for-the-badge&logo=facebook)](https://www.facebook.com/iris.framework)

## 🤖 ابزارهای هوش مصنوعی برای Iris

دو ابزار هوش مصنوعی این فریم‌ورک را به‌طور خاص می‌شناسند، فراتر از آنچه یک مدل عمومی دربارهٔ Go می‌داند.

[ویکی هوش مصنوعی زنده](https://codewiki.google/github.com/kataras/iris) رایگان است و در مرورگر اجرا می‌شود: معماری را مرور کنید یا از چت آن دربارهٔ کدبیس بپرسید.

[Plexon AI](https://plexon.ai) یک دستیار دسکتاپ برای Windows، macOS و Linux است که نویسندهٔ Iris آن را ساخته است. اسکیل Iris ([iris-go.com/ai](https://iris-go.com/ai/)) را به‌صورت داخلی همراه دارد: API نسخهٔ v14، زنجیرهٔ builder، هلپرهای `rest`، نگاشت خطاها، SDK احراز هویت و الگوهای تست، در بیست و یک سند مرجع که دستیار هنگام نوشتن کد شما و بر حسب نیاز آن‌ها را می‌خواند. اسکیل را از پنل Skills فعال کنید یا [پرسونای Software Developer](https://plexon.ai/personas/software-developer/) را نصب کنید که آن را همراه با بقیهٔ ابزارهای توسعه فعال می‌کند. [Plexon AI را دانلود کنید](https://plexon.ai/download/).

---

برای حامیان، مشارکت در توسعه، سیاست امنیتی و مجوز، [README انگلیسی](README.md) را ببینید.

این پروژه تحت [مجوز BSD 3-clause](LICENSE) منتشر شده است، درست مانند خود پروژهٔ Go.
