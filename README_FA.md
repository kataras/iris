<div dir="rtl">
    
## خبرها
    
> این شاخه تحت توسعه است. برای رفتن به شاخه نسخه بعدی [v12.2.0](HISTORY.md#Next) یا اگر به دنبال یک انتشار پایدار هستید, به جای آن به شاخه [v12.1.8 branch](https://github.com/kataras/iris/tree/v12.1.8) مراجعه کنید.
    
> ![](https://iris-go.com/images/cli.png) همین امروز برنامه رسمی [Iris Command Line Interface](https://github.com/kataras/iris-cli) را امتحان کنید.

> با توجه به بالا بودن حجم کار، ممکن است در پاسخ به [سوالات](https://github.com/kataras/iris/issues) شما تاخیری وجود داشته باشد.

# چارچوب وب آیریس
    
[![build status](https://img.shields.io/github/workflow/status/kataras/iris/CI/master?style=for-the-badge)](https://github.com/kataras/iris/actions) [![FOSSA Status](https://img.shields.io/badge/LICENSE%20SCAN-PASSING❤️-CD2956?style=for-the-badge&logo=fossa)](https://app.fossa.io/projects/git%2Bgithub.com%2Fkataras%2Firis?ref=badge_shield)<!--[![report card](https://img.shields.io/badge/report%20card-a%2B-ff3333.svg?style=for-the-badge)](https://goreportcard.com/report/github.com/kataras/iris)--><!--[![godocs](https://img.shields.io/badge/go-%20docs-488AC7.svg?style=for-the-badge)](https://pkg.go.dev/github.com/kataras/iris/v12@v12.2.0)--> [![view examples](https://img.shields.io/badge/learn%20by-examples-0C8EC5.svg?style=for-the-badge&logo=go)](https://github.com/kataras/iris/tree/master/_examples) [![chat](https://img.shields.io/gitter/room/iris_go/community.svg?color=7E18DD&logo=gitter&style=for-the-badge)](https://gitter.im/iris_go/community)<!--[![donate on PayPal](https://img.shields.io/badge/support-PayPal-blue.svg?style=for-the-badge)](https://iris-go.com/donate)--><!-- [![release](https://img.shields.io/badge/release%20-v12.0-0077b3.svg?style=for-the-badge)](https://github.com/kataras/iris/releases) -->
    
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
> !برای اطلاعات بیشتر ، [مثال های مسیریابی](https://github.com/kataras/iris/blob/master/_examples/routing) را بخوانید

</details>

<details><summary>Handler with custom input and output arguments</summary>

[![https://github.com/kataras/iris/blob/master/_examples/dependency-injection/basic/main.go](https://user-images.githubusercontent.com/22900943/105253731-b8db6d00-5b88-11eb-90c1-0c92a5581c86.png)](https://twitter.com/iris_framework/status/1234783655408668672)

> اگر برایتان جالب بود [مثال های دیگری](https://github.com/kataras/iris/blob/master/_examples/dependency-injection) را مطالعه کنید

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
    
<p>
  <a href="https://github.com/paulxu21"><img src="https://avatars1.githubusercontent.com/u/6261758?v=4" alt ="Paul Xu" title="paulxu21" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/DavidShaw"><img src="https://avatars1.githubusercontent.com/u/356970?v=4" alt ="David Shaw" title="DavidShaw" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/lingyingtan"><img src="https://avatars1.githubusercontent.com/u/15610136?v=4" alt ="Stone Travel" title="lingyingtan" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/Laotanling"><img src="https://avatars1.githubusercontent.com/u/28570289?v=4" alt ="Tan" title="Laotanling" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/acdias"><img src="https://avatars1.githubusercontent.com/u/11966653?v=4" alt ="Andre Dias" title="acdias" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/rfunix"><img src="https://avatars1.githubusercontent.com/u/6026357?v=4" alt ="Rafael Francischini" title="rfunix" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/liheyuan"><img src="https://avatars1.githubusercontent.com/u/776423?v=4" alt ="Heyuan Li" title="liheyuan" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/RainerGevers"><img src="https://avatars1.githubusercontent.com/u/32453861?v=4" alt ="Rainer Gevers" title="RainerGevers" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/shadowfiga"><img src="https://avatars1.githubusercontent.com/u/42721390?v=4" alt ="Matic Zarnec" title="shadowfiga" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/knavels"><img src="https://avatars1.githubusercontent.com/u/57287952?v=4" alt ="Navid Dezashibi" title="knavels" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/rxrw"><img src="https://avatars1.githubusercontent.com/u/9566402?v=4" alt ="Sky Lee" title="rxrw" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/rbondi"><img src="https://avatars1.githubusercontent.com/u/81764?v=4" alt ="Richard Bondi" title="rbondi" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/aprinslo1"><img src="https://avatars1.githubusercontent.com/u/711650?v=4" alt ="Anthonius Prinslo" title="aprinslo1" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/wofka72"><img src="https://avatars1.githubusercontent.com/u/10855340?v=4" alt ="Vladimir" title="wofka72" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/GeorgeFourikis"><img src="https://avatars1.githubusercontent.com/u/17906313?v=4" alt ="George Fourikis" title="GeorgeFourikis" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/mblandr"><img src="https://avatars1.githubusercontent.com/u/42862020?v=4" alt ="Александр Лебединский" title="mblandr" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/cpp2go"><img src="https://avatars1.githubusercontent.com/u/12148026?v=4" alt ="Li Yang" title="cpp2go" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/lafayetteDan"><img src="https://avatars1.githubusercontent.com/u/26064396?v=4" alt ="Qianyu Zhou" title="lafayetteDan" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/anilpdv"><img src="https://avatars1.githubusercontent.com/u/32708402?v=4" alt ="anilpdv" title="anilpdv" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/baoch254"><img src="https://avatars1.githubusercontent.com/u/74555344?v=4" alt ="CAO HOAI BAO" title="baoch254" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/hdezoscar93"><img src="https://avatars1.githubusercontent.com/u/21270107?v=4" alt ="Oscar Hernandez" title="hdezoscar93" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/blackHoleNgc1277"><img src="https://avatars1.githubusercontent.com/u/41342763?v=4" alt ="Gerard Lancea" title="blackHoleNgc1277" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/Neulhan"><img src="https://avatars1.githubusercontent.com/u/52434903?v=4" alt ="neulhan" title="Neulhan" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/kana99"><img src="https://avatars1.githubusercontent.com/u/3714069?v=4" alt ="xushiquan" title="kana99" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/mattbowen"><img src="https://avatars1.githubusercontent.com/u/46803?v=4" alt ="Matt" title="mattbowen" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/lpintes"><img src="https://avatars1.githubusercontent.com/u/2546783?v=4" alt ="Ľuboš Pinteš" title="lpintes" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/mmckeen75"><img src="https://avatars1.githubusercontent.com/u/49529489?v=4" alt ="Leighton McKeen" title="mmckeen75" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/lauweliam"><img src="https://avatars1.githubusercontent.com/u/4064517?v=4" alt ="Weliam" title="lauweliam" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/galois-tnp"><img src="https://avatars1.githubusercontent.com/u/41128011?v=4" alt ="simranjit singh" title="galois-tnp" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/gnosthi"><img src="https://avatars1.githubusercontent.com/u/17650528?v=4" alt ="Kenneth Jordan" title="gnosthi" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/ndimorle"><img src="https://avatars1.githubusercontent.com/u/76732415?v=4" alt ="Morlé Koudeka" title="ndimorle" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/rsousacode"><img src="https://avatars1.githubusercontent.com/u/34067397?v=4" alt ="Rui" title="rsousacode" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/carlos-enginner"><img src="https://avatars1.githubusercontent.com/u/59775876?v=4" alt ="Carlos Augusto" title="carlos-enginner" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/motogo"><img src="https://avatars1.githubusercontent.com/u/1704958?v=4" alt ="Horst Ender" title="motogo" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/remopavithran"><img src="https://avatars1.githubusercontent.com/u/50388068?v=4" alt ="Pavithran" title="remopavithran" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/mulyawansentosa"><img src="https://avatars1.githubusercontent.com/u/29946673?v=4" alt ="MULYAWAN SENTOSA" title="mulyawansentosa" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/TianJIANG"><img src="https://avatars1.githubusercontent.com/u/158459?v=4" alt ="KIT UNITED" title="TianJIANG" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/rhernandez-itemsoft"><img src="https://avatars1.githubusercontent.com/u/4327356?v=4" alt ="Ricardo Hernandez Lopez" title="rhernandez-itemsoft" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/ChinChuanKuo"><img src="https://avatars1.githubusercontent.com/u/11756978?v=4" alt ="ChinChuanKuo" title="ChinChuanKuo" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/nikharsaxena"><img src="https://avatars1.githubusercontent.com/u/8684362?v=4" alt ="Nikhar Saxena" title="nikharsaxena" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/fenriz07"><img src="https://avatars1.githubusercontent.com/u/9199380?v=4" alt ="Servio Zambrano" title="fenriz07" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/NA"><img src="https://avatars1.githubusercontent.com/u/1600?v=4" alt ="Nate Anderson" title="NA" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/claudemuller"><img src="https://avatars1.githubusercontent.com/u/8104894?v=4" alt ="Claude Muller" title="claudemuller" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/marcmmx"><img src="https://avatars1.githubusercontent.com/u/7670546?v=4" alt ="Marco Moeser" title="marcmmx" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/sankethpb"><img src="https://avatars1.githubusercontent.com/u/16034868?v=4" alt ="Sanketh P B" title="sankethpb" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/vuhoanglam"><img src="https://avatars1.githubusercontent.com/u/59502855?v=4" alt ="Vu Hoang Lam" title="vuhoanglam" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/dtrifonov"><img src="https://avatars1.githubusercontent.com/u/1520118?v=4" alt ="Dimitar Trifonov" title="dtrifonov" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/midhubalan"><img src="https://avatars1.githubusercontent.com/u/13059634?v=4" alt ="Midhubalan Balasubramanian" title="midhubalan" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/tuxaanand"><img src="https://avatars1.githubusercontent.com/u/9750371?v=4" alt ="AANAND NATARAJAN" title="tuxaanand" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/edsongley"><img src="https://avatars1.githubusercontent.com/u/35545454?v=4" alt ="Edsongley Almeida" title="edsongley" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/ganben"><img src="https://avatars1.githubusercontent.com/u/10101347?v=4" alt ="ganben" title="ganben" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/tejzpr"><img src="https://avatars1.githubusercontent.com/u/2813811?v=4" alt ="Tejus Pratap" title="tejzpr" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/BlackHole1"><img src="https://avatars1.githubusercontent.com/u/8198408?v=4" alt ="cui hexiang" title="BlackHole1" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/wangbl11"><img src="https://avatars1.githubusercontent.com/u/14358532?v=4" alt ="tinawang" title="wangbl11" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/juanxme"><img src="https://avatars1.githubusercontent.com/u/661043?v=4" alt ="Juan David Parra Pimiento" title="juanxme" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/andychongyz"><img src="https://avatars1.githubusercontent.com/u/12697240?v=4" alt ="Andy Chong Ying Zhi" title="andychongyz" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/KevinZhouRafael"><img src="https://avatars1.githubusercontent.com/u/16298046?v=4" alt ="Kevin Zhou" title="KevinZhouRafael" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/mizzlespot"><img src="https://avatars1.githubusercontent.com/u/2654538?v=4" alt ="Jasper" title="mizzlespot" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/wiener01mu"><img src="https://avatars1.githubusercontent.com/u/41128011?v=4" alt ="Simranjit Singh" title="wiener01mu" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/theantichris"><img src="https://avatars1.githubusercontent.com/u/1486502?v=4" alt ="Christopher Lamm" title="theantichris" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/L-M-Sherlock"><img src="https://avatars1.githubusercontent.com/u/32575846?v=4" alt ="叶峻峣" title="L-M-Sherlock" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/tsailiting"><img src="https://avatars1.githubusercontent.com/u/48909556?v=4" alt ="TSAI LI TING" title="tsailiting" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/TonyZhu"><img src="https://avatars1.githubusercontent.com/u/677477?v=4" alt ="zhutao" title="TonyZhu" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/goten002"><img src="https://avatars1.githubusercontent.com/u/5025060?v=4" alt ="George Alexiou" title="goten002" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/jazar17"><img src="https://avatars1.githubusercontent.com/u/1813513?v=4" alt ="Jobert Azares" title="jazar17" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/nguyentamvinhlong"><img src="https://avatars1.githubusercontent.com/u/1875916?v=4" alt ="Tam Nguyen" title="nguyentamvinhlong" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/vguhesan"><img src="https://avatars1.githubusercontent.com/u/193960?v=4" alt ="
Venkatt Guhesan" title="vguhesan" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/Cesar"><img src="https://avatars1.githubusercontent.com/u/1581870?v=4" alt ="Anibal C C Budaye" title="Cesar" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/unixedia"><img src="https://avatars1.githubusercontent.com/u/70646128?v=4" alt ="ARAN ROKA" title="unixedia" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/ky2s"><img src="https://avatars1.githubusercontent.com/u/19502125?v=4" alt ="Valentine" title="ky2s" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/CSRaghunandan"><img src="https://avatars1.githubusercontent.com/u/5226809?v=4" alt ="Chakravarthy Raghunandan" title="CSRaghunandan" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/maxbertinetti"><img src="https://avatars1.githubusercontent.com/u/26814295?v=4" alt ="Massimiliano Bertinetti" title="maxbertinetti" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/HieuLsw"><img src="https://avatars1.githubusercontent.com/u/1675478?v=4" alt ="Hieu Trinh" title="HieuLsw" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/jingtianfeng"><img src="https://avatars1.githubusercontent.com/u/19503202?v=4" alt ="J.T. Feng" title="jingtianfeng" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/leki75"><img src="https://avatars1.githubusercontent.com/u/9675379?v=4" alt ="Gabor Lekeny" title="leki75" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/lihaotian0607"><img src="https://avatars1.githubusercontent.com/u/32523475?v=4" alt ="LiHaotian" title="lihaotian0607" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/Little-YangYang"><img src="https://avatars1.githubusercontent.com/u/10755202?v=4" alt ="Muyang Li" title="Little-YangYang" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/tuhao1020"><img src="https://avatars1.githubusercontent.com/u/26807520?v=4" alt ="Hao Tu" title="tuhao1020" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/CetinBasoz"><img src="https://avatars1.githubusercontent.com/u/3152637?v=4" alt ="Cetin Basoz" title="CetinBasoz" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/hazmi-e205"><img src="https://avatars1.githubusercontent.com/u/12555465?v=4" alt ="Hazmi Amalul" title="hazmi-e205" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/remyDeme"><img src="https://avatars1.githubusercontent.com/u/22757039?v=4" alt ="Rémy Deme" title="remyDeme" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/vincent-li"><img src="https://avatars1.githubusercontent.com/u/765470?v=4" alt ="Vincent Li" title="vincent-li" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/mtrense"><img src="https://avatars1.githubusercontent.com/u/1008285?v=4" alt ="Max Trense" title="mtrense" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/MatejLach"><img src="https://avatars1.githubusercontent.com/u/531930?v=4" alt ="Matej Lach" title="MatejLach" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/sumjoe"><img src="https://avatars1.githubusercontent.com/u/32655210?v=4" alt ="Joseph De Paola" title="sumjoe" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/AlbinoGeek"><img src="https://avatars1.githubusercontent.com/u/1910461?v=4" alt ="Damon Blais" title="AlbinoGeek" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/LYF123123"><img src="https://avatars1.githubusercontent.com/u/33317812?v=4" alt ="陆 轶丰" title="LYF123123" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/xiaozhuai"><img src="https://avatars1.githubusercontent.com/u/4773701?v=4" alt ="Weihang Ding" title="xiaozhuai" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/fangli"><img src="https://avatars1.githubusercontent.com/u/3032639?v=4" alt ="Li Fang" title="fangli" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/TechMaster"><img src="https://avatars1.githubusercontent.com/u/1491686?v=4" alt ="TechMaster" title="TechMaster" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/lensesio"><img src="https://avatars1.githubusercontent.com/u/11728472?v=4" alt ="lenses.io" title="lensesio" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/celsosz"><img src="https://avatars1.githubusercontent.com/u/3466493?v=4" alt ="Celso Souza" title="celsosz" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/altafino"><img src="https://avatars1.githubusercontent.com/u/24539467?v=4" alt ="Altafino" title="altafino" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/thomasfr"><img src="https://avatars1.githubusercontent.com/u/287432?v=4" alt ="Thomas Fritz" title="thomasfr" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/hengestone"><img src="https://avatars1.githubusercontent.com/u/362587?v=4" alt ="Conrad Steenberg" title="hengestone" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/se77en"><img src="https://avatars1.githubusercontent.com/u/1468284?v=4" alt ="Damon Zhao" title="se77en" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/International"><img src="https://avatars1.githubusercontent.com/u/1022918?v=4" alt ="George Opritescu" title="International" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/Juanses"><img src="https://avatars1.githubusercontent.com/u/6137970?v=4" alt ="Juanses" title="Juanses" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/ansrivas"><img src="https://avatars1.githubusercontent.com/u/1695056?v=4" alt ="Ankur Srivastava" title="ansrivas" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/lexrus"><img src="https://avatars1.githubusercontent.com/u/219689?v=4" alt ="Lex Tang" title="lexrus" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/li3p"><img src="https://avatars1.githubusercontent.com/u/55519?v=4" alt ="li3p" title="li3p" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
</p>
    
## اموزش آیریس
    
### ساخت یک پروژه جدید

</div>
    
```sh
$ mkdir myapp
$ cd myapp
$ go mod init myapp
$ go get github.com/kataras/iris/v12@master # or @v12.2.0-alpha8
```

<div dir="rtl">
<summary>نصب بر روی پروژه موجود</summary>
</div>

```sh
$ cd myapp
$ go get github.com/kataras/iris/v12@master
```

<div dir="rtl">
<summary>نصب با پرونده go.mod</summary>
</div>

```txt
module myapp

go 1.17

require github.com/kataras/iris/v12 master
```
![](https://www.iris-go.com/images/gifs/install-create-iris.gif)

<div dir="rtl">
آیریس شامل مستندات گسترده و کاملی است که کار با چارچوب را آسان می کند.

> [مستندات](https://www.iris-go.com/docs)
    
برای اطلاعات بیشتر در مورد اسناد فنی می توانید به مستندات اصلی ما مراجعه کنید. 

> [مستندات اصلی](https://pkg.go.dev/github.com/kataras/iris/v12@master)
    
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
  
