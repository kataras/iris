<!-- [![Black Lives Matter](https://iris-go.com/images/blacklivesmatter_banner.png)](https://support.eji.org/give/153413/#!/donation/checkout)


# News

> This is the under-**development branch** - contains the latest and greatest features. Stay tuned for the upcoming release [v12.2.0](HISTORY.md#Next). Looking for a more stable release? Head over to the [v12.1.8 branch](https://github.com/kataras/iris/tree/v12.1.8) instead.
>
> ![](https://iris-go.com/images/cli.png) Try the official [Iris Command Line Interface](https://github.com/kataras/iris-cli) today!

> Due to the large workload, there may be delays in answering your [questions](https://github.com/kataras/iris/issues). -->

<!-- ![](https://iris-go.com/images/release.png) Iris version **12.1.8** has been [released](HISTORY.md#su-16-february-2020--v1218)! -->

# Iris Web Framework <a href="README_GR.md"><img width="20px" src="https://iris-go.com/images/flag-greece.svg" /></a> <a href="README_FR.md"><img width="20px" src="https://iris-go.com/images/flag-france.svg" /></a> <a href="README_ZH.md"><img width="20px" src="https://iris-go.com/images/flag-china.svg" /></a> <a href="README_ES.md"><img width="20px" src="https://iris-go.com/images/flag-spain.png" /></a> <a href="README_FA.md"><img width="20px" src="https://iris-go.com/images/flag-iran.svg" /></a> <a href="README_RU.md"><img width="20px" src="https://iris-go.com/images/flag-russia.svg" /></a> <a href="README_KO.md"><img width="20px" src="https://iris-go.com/images/flag-south-korea.svg?v=12" /></a>

[![build status](https://img.shields.io/github/workflow/status/kataras/iris/CI/master?style=for-the-badge)](https://github.com/kataras/iris/actions) [![view examples](https://img.shields.io/badge/examples%20-253-a83adf.svg?style=for-the-badge&logo=go)](https://github.com/kataras/iris/tree/master/_examples) [![chat](https://img.shields.io/gitter/room/iris_go/community.svg?color=cc2b5e&logo=gitter&style=for-the-badge)](https://gitter.im/iris_go/community) <!--[![FOSSA Status](https://img.shields.io/badge/LICENSE%20SCAN-PASSING❤️-CD2956?style=for-the-badge&logo=fossa)](https://app.fossa.io/projects/git%2Bgithub.com%2Fkataras%2Firis?ref=badge_shield)--> [![donate](https://img.shields.io/badge/support-Iris-blue.svg?style=for-the-badge&logo=paypal)](https://iris-go.com/donate) <!--[![report card](https://img.shields.io/badge/report%20card-a%2B-ff3333.svg?style=for-the-badge)](https://goreportcard.com/report/github.com/kataras/iris)--><!--[![godocs](https://img.shields.io/badge/go-%20docs-488AC7.svg?style=for-the-badge)](https://pkg.go.dev/github.com/kataras/iris/v12@v12.2.0)--> <!-- [![release](https://img.shields.io/badge/release%20-v12.0-0077b3.svg?style=for-the-badge)](https://github.com/kataras/iris/releases) -->

<!-- <a href="https://iris-go.com"> <img align="right" src="https://iris-go.com/images/logo-w169.png"></a> -->

Iris is a fast, simple yet fully featured and very efficient web framework for Go.

It provides a beautifully expressive and easy to use foundation for your next website or API.

```go
package main

import "github.com/kataras/iris/v12"

func main() {
	app := iris.New()
	app.Use(iris.Compression)

	app.Get("/", func(ctx iris.Context) {
		ctx.HTML("Hello <strong>%s</strong>!", "World")
	})

	app.Listen(":8080")
}
```

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

> Read the [routing examples](https://github.com/kataras/iris/blob/master/_examples/routing) for more!

</details>

<details><summary>Handler with custom input and output arguments</summary>

[![https://github.com/kataras/iris/blob/master/_examples/dependency-injection/basic/main.go](https://user-images.githubusercontent.com/22900943/105253731-b8db6d00-5b88-11eb-90c1-0c92a5581c86.png)](https://twitter.com/iris_framework/status/1234783655408668672)

> Interesting? Read the [examples](https://github.com/kataras/iris/blob/master/_examples/dependency-injection).

</details>

<details><summary>Party Controller (NEW)</summary>

> Head over to the [full running example](https://github.com/kataras/iris/blob/master/_examples/routing/party-controller)!

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



Learn what [others saying about Iris](https://www.iris-go.com/#review) and **[star](https://github.com/kataras/iris/stargazers)** this open-source project to support its potentials.

[![](https://iris-go.com/images/reviews.gif)](https://iris-go.com/testimonials/)

[![Benchmarks: Jul 18, 2020 at 10:46am (UTC)](https://iris-go.com/images/benchmarks.svg)](https://github.com/kataras/server-benchmarks)

## 👑 <a href="https://iris-go.com/donate">Supporters</a>

With your help, we can improve Open Source web development for everyone!

> Donations from **China** are now accepted!

<p>
  <a href="https://github.com/lensesio"><img src="https://avatars1.githubusercontent.com/u/11728472?v=4" alt="lensesio" title="lensesio" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/trading-peter"><img src="https://avatars1.githubusercontent.com/u/11567985?v=4" alt="trading-peter" title="trading-peter" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/basilarchia"><img src="https://avatars1.githubusercontent.com/u/926033?v=4" alt="basilarchia" title="basilarchia" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/xiaozhuai"><img src="https://avatars1.githubusercontent.com/u/4773701?v=4" alt="xiaozhuai" title="xiaozhuai" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/AlbinoGeek"><img src="https://avatars1.githubusercontent.com/u/1910461?v=4" alt="AlbinoGeek" title="AlbinoGeek" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/celsosz"><img src="https://avatars1.githubusercontent.com/u/3466493?v=4" alt="celsosz" title="celsosz" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/TechMaster"><img src="https://avatars1.githubusercontent.com/u/1491686?v=4" alt="TechMaster" title="TechMaster" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/altafino"><img src="https://avatars1.githubusercontent.com/u/24539467?v=4" alt="altafino" title="altafino" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/hengestone"><img src="https://avatars1.githubusercontent.com/u/362587?v=4" alt="hengestone" title="hengestone" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/thomasfr"><img src="https://avatars1.githubusercontent.com/u/287432?v=4" alt="thomasfr" title="thomasfr" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/International"><img src="https://avatars1.githubusercontent.com/u/1022918?v=4" alt="International" title="International" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/Juanses"><img src="https://avatars1.githubusercontent.com/u/6137970?v=4" alt="Juanses" title="Juanses" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/ansrivas"><img src="https://avatars1.githubusercontent.com/u/1695056?v=4" alt="ansrivas" title="ansrivas" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/draFWM"><img src="https://avatars1.githubusercontent.com/u/5765340?v=4" alt="draFWM" title="draFWM" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/gf3"><img src="https://avatars1.githubusercontent.com/u/18397?v=4" alt="gf3" title="gf3" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/lexrus"><img src="https://avatars1.githubusercontent.com/u/219689?v=4" alt="lexrus" title="lexrus" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/li3p"><img src="https://avatars1.githubusercontent.com/u/55519?v=4" alt="li3p" title="li3p" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/se77en"><img src="https://avatars1.githubusercontent.com/u/1468284?v=4" alt="se77en" title="se77en" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/sumjoe"><img src="https://avatars1.githubusercontent.com/u/32655210?v=4" alt="sumjoe" title="sumjoe" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/vincent-li"><img src="https://avatars1.githubusercontent.com/u/765470?v=4" alt="vincent-li" title="vincent-li" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/sascha11110"><img src="https://avatars1.githubusercontent.com/u/15168372?v=4" alt="sascha11110" title="sascha11110" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/derReineke"><img src="https://avatars1.githubusercontent.com/u/35681013?v=4" alt="derReineke" title="derReineke" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/Sirisap22"><img src="https://avatars1.githubusercontent.com/u/58851659?v=4" alt="Sirisap22" title="Sirisap22" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/clacroix"><img src="https://avatars1.githubusercontent.com/u/611064?v=4" alt="clacroix" title="clacroix" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/ixalender"><img src="https://avatars1.githubusercontent.com/u/877376?v=4" alt="ixalender" title="ixalender" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/mubariz-ahmed"><img src="https://avatars1.githubusercontent.com/u/18215455?v=4" alt="mubariz-ahmed" title="mubariz-ahmed" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/rodrigoghm"><img src="https://avatars1.githubusercontent.com/u/66917643?v=4" alt="rodrigoghm" title="rodrigoghm" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/Cesar"><img src="https://avatars1.githubusercontent.com/u/1581870?v=4" alt="Cesar" title="Cesar" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/DavidShaw"><img src="https://avatars1.githubusercontent.com/u/356970?v=4" alt="DavidShaw" title="DavidShaw" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/DmarshalTU"><img src="https://avatars1.githubusercontent.com/u/59089266?v=4" alt="DmarshalTU" title="DmarshalTU" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/IwateKyle"><img src="https://avatars1.githubusercontent.com/u/658799?v=4" alt="IwateKyle" title="IwateKyle" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/Little-YangYang"><img src="https://avatars1.githubusercontent.com/u/10755202?v=4" alt="Little-YangYang" title="Little-YangYang" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/coderperu"><img src="https://avatars1.githubusercontent.com/u/68706957?v=4" alt="coderperu" title="coderperu" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/cshum"><img src="https://avatars1.githubusercontent.com/u/293790?v=4" alt="cshum" title="cshum" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/dtrifonov"><img src="https://avatars1.githubusercontent.com/u/1520118?v=4" alt="dtrifonov" title="dtrifonov" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/ichenhe"><img src="https://avatars1.githubusercontent.com/u/10266066?v=4" alt="ichenhe" title="ichenhe" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/icibiri"><img src="https://avatars1.githubusercontent.com/u/32684966?v=4" alt="icibiri" title="icibiri" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/jingtianfeng"><img src="https://avatars1.githubusercontent.com/u/19503202?v=4" alt="jingtianfeng" title="jingtianfeng" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/kilarusravankumar"><img src="https://avatars1.githubusercontent.com/u/13055113?v=4" alt="kilarusravankumar" title="kilarusravankumar" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/leandrobraga"><img src="https://avatars1.githubusercontent.com/u/506699?v=4" alt="leandrobraga" title="leandrobraga" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/lfbos"><img src="https://avatars1.githubusercontent.com/u/5703286?v=4" alt="lfbos" title="lfbos" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/lpintes"><img src="https://avatars1.githubusercontent.com/u/2546783?v=4" alt="lpintes" title="lpintes" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/macropas"><img src="https://avatars1.githubusercontent.com/u/7488502?v=4" alt="macropas" title="macropas" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/marcmmx"><img src="https://avatars1.githubusercontent.com/u/7670546?v=4" alt="marcmmx" title="marcmmx" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/mihado"><img src="https://avatars1.githubusercontent.com/u/940981?v=4" alt="mihado" title="mihado" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/mmckeen75"><img src="https://avatars1.githubusercontent.com/u/49529489?v=4" alt="mmckeen75" title="mmckeen75" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/olaf-lexemo"><img src="https://avatars1.githubusercontent.com/u/51406599?v=4" alt="olaf-lexemo" title="olaf-lexemo" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/pitexplore"><img src="https://avatars1.githubusercontent.com/u/11956562?v=4" alt="pitexplore" title="pitexplore" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/pr123"><img src="https://avatars1.githubusercontent.com/u/23333176?v=4" alt="pr123" title="pr123" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/sankethpb"><img src="https://avatars1.githubusercontent.com/u/16034868?v=4" alt="sankethpb" title="sankethpb" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/saz59"><img src="https://avatars1.githubusercontent.com/u/9706793?v=4" alt="saz59" title="saz59" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/shadowfiga"><img src="https://avatars1.githubusercontent.com/u/42721390?v=4" alt="shadowfiga" title="shadowfiga" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/skurtz97"><img src="https://avatars1.githubusercontent.com/u/71720714?v=4" alt="skurtz97" title="skurtz97" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/srinivasganti"><img src="https://avatars1.githubusercontent.com/u/2057165?v=4" alt="srinivasganti" title="srinivasganti" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/tuhao1020"><img src="https://avatars1.githubusercontent.com/u/26807520?v=4" alt="tuhao1020" title="tuhao1020" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/wahyuief"><img src="https://avatars1.githubusercontent.com/u/20138856?v=4" alt="wahyuief" title="wahyuief" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/xvalen"><img src="https://avatars1.githubusercontent.com/u/2307513?v=4" alt="xvalen" title="xvalen" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/xytis"><img src="https://avatars1.githubusercontent.com/u/78025?v=4" alt="xytis" title="xytis" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/ElNovi"><img src="https://avatars1.githubusercontent.com/u/14199592?v=4" alt="ElNovi" title="ElNovi" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/KKP4"><img src="https://avatars1.githubusercontent.com/u/24271790?v=4" alt="KKP4" title="KKP4" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/Lernakow"><img src="https://avatars1.githubusercontent.com/u/46821665?v=4" alt="Lernakow" title="Lernakow" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/Major2828"><img src="https://avatars1.githubusercontent.com/u/19783402?v=4" alt="Major2828" title="Major2828" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/MatejLach"><img src="https://avatars1.githubusercontent.com/u/531930?v=4" alt="MatejLach" title="MatejLach" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/odas0r"><img src="https://avatars1.githubusercontent.com/u/32167770?v=4" alt="odas0r" title="odas0r" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/syrm"><img src="https://avatars1.githubusercontent.com/u/155406?v=4" alt="syrm" title="syrm" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/ukitzmann"><img src="https://avatars1.githubusercontent.com/u/153834?v=4" alt="ukitzmann" title="ukitzmann" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/aprinslo1"><img src="https://avatars1.githubusercontent.com/u/711650?v=4" alt="aprinslo1" title="aprinslo1" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/kyoukhana"><img src="https://avatars1.githubusercontent.com/u/756849?v=4" alt="kyoukhana" title="kyoukhana" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/mark2b"><img src="https://avatars1.githubusercontent.com/u/539063?v=4" alt="mark2b" title="mark2b" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/siriushaha"><img src="https://avatars1.githubusercontent.com/u/7924311?v=4" alt="siriushaha" title="siriushaha" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/spazzymoto"><img src="https://avatars1.githubusercontent.com/u/2951012?v=4" alt="spazzymoto" title="spazzymoto" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/kukaki"><img src="https://avatars1.githubusercontent.com/u/4849535?v=4" alt="kukaki" title="kukaki" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/oshirokazuhide"><img src="https://avatars1.githubusercontent.com/u/89958891?v=4" alt="oshirokazuhide" title="oshirokazuhide" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/t6tg"><img src="https://avatars1.githubusercontent.com/u/33445861?v=4" alt="t6tg" title="t6tg" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/AwsIT"><img src="https://avatars1.githubusercontent.com/u/40926862?v=4" alt="AwsIT" title="AwsIT" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/BlackHole1"><img src="https://avatars1.githubusercontent.com/u/8198408?v=4" alt="BlackHole1" title="BlackHole1" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/Jude-X"><img src="https://avatars1.githubusercontent.com/u/66228813?v=4" alt="Jude-X" title="Jude-X" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/KevinZhouRafael"><img src="https://avatars1.githubusercontent.com/u/16298046?v=4" alt="KevinZhouRafael" title="KevinZhouRafael" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/Laotanling"><img src="https://avatars1.githubusercontent.com/u/28570289?v=4" alt="Laotanling" title="Laotanling" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/MihaiPopescu1985"><img src="https://avatars1.githubusercontent.com/u/34679869?v=4" alt="MihaiPopescu1985" title="MihaiPopescu1985" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/Neulhan"><img src="https://avatars1.githubusercontent.com/u/52434903?v=4" alt="Neulhan" title="Neulhan" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/NguyenPhuoc"><img src="https://avatars1.githubusercontent.com/u/11747677?v=4" alt="NguyenPhuoc" title="NguyenPhuoc" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/SamuelNeves"><img src="https://avatars1.githubusercontent.com/u/10797137?v=4" alt="SamuelNeves" title="SamuelNeves" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/TianJIANG"><img src="https://avatars1.githubusercontent.com/u/158459?v=4" alt="TianJIANG" title="TianJIANG" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/Ubun1"><img src="https://avatars1.githubusercontent.com/u/13261595?v=4" alt="Ubun1" title="Ubun1" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/acdias"><img src="https://avatars1.githubusercontent.com/u/11966653?v=4" alt="acdias" title="acdias" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/agent3bood"><img src="https://avatars1.githubusercontent.com/u/771902?v=4" alt="agent3bood" title="agent3bood" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/b2cbd"><img src="https://avatars1.githubusercontent.com/u/6870050?v=4" alt="b2cbd" title="b2cbd" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/baoch254"><img src="https://avatars1.githubusercontent.com/u/74555344?v=4" alt="baoch254" title="baoch254" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/bastengao"><img src="https://avatars1.githubusercontent.com/u/785335?v=4" alt="bastengao" title="bastengao" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/bunnycodego"><img src="https://avatars1.githubusercontent.com/u/81451316?v=4" alt="bunnycodego" title="bunnycodego" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/carlos-enginner"><img src="https://avatars1.githubusercontent.com/u/59775876?v=4" alt="carlos-enginner" title="carlos-enginner" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/civicwar"><img src="https://avatars1.githubusercontent.com/u/1858104?v=4" alt="civicwar" title="civicwar" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/cnzhangquan"><img src="https://avatars1.githubusercontent.com/u/5462876?v=4" alt="cnzhangquan" title="cnzhangquan" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/edwindna2"><img src="https://avatars1.githubusercontent.com/u/5441354?v=4" alt="edwindna2" title="edwindna2" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/fenriz07"><img src="https://avatars1.githubusercontent.com/u/9199380?v=4" alt="fenriz07" title="fenriz07" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/gnosthi"><img src="https://avatars1.githubusercontent.com/u/17650528?v=4" alt="gnosthi" title="gnosthi" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/goten002"><img src="https://avatars1.githubusercontent.com/u/5025060?v=4" alt="goten002" title="goten002" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/guanzi008"><img src="https://avatars1.githubusercontent.com/u/20619190?v=4" alt="guanzi008" title="guanzi008" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/hdezoscar93"><img src="https://avatars1.githubusercontent.com/u/21270107?v=4" alt="hdezoscar93" title="hdezoscar93" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/hzxd"><img src="https://avatars1.githubusercontent.com/u/3376231?v=4" alt="hzxd" title="hzxd" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/iantuan"><img src="https://avatars1.githubusercontent.com/u/4869968?v=4" alt="iantuan" title="iantuan" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/kana99"><img src="https://avatars1.githubusercontent.com/u/3714069?v=4" alt="kana99" title="kana99" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/khasanovrs"><img src="https://avatars1.githubusercontent.com/u/6076966?v=4" alt="khasanovrs" title="khasanovrs" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/knavels"><img src="https://avatars1.githubusercontent.com/u/57287952?v=4" alt="knavels" title="knavels" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/leki75"><img src="https://avatars1.githubusercontent.com/u/9675379?v=4" alt="leki75" title="leki75" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/liheyuan"><img src="https://avatars1.githubusercontent.com/u/776423?v=4" alt="liheyuan" title="liheyuan" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/lingyingtan"><img src="https://avatars1.githubusercontent.com/u/15610136?v=4" alt="lingyingtan" title="lingyingtan" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/lipatti"><img src="https://avatars1.githubusercontent.com/u/38935867?v=4" alt="lipatti" title="lipatti" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/mattbowen"><img src="https://avatars1.githubusercontent.com/u/46803?v=4" alt="mattbowen" title="mattbowen" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/mizzlespot"><img src="https://avatars1.githubusercontent.com/u/2654538?v=4" alt="mizzlespot" title="mizzlespot" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/mnievesco"><img src="https://avatars1.githubusercontent.com/u/78430169?v=4" alt="mnievesco" title="mnievesco" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/motogo"><img src="https://avatars1.githubusercontent.com/u/1704958?v=4" alt="motogo" title="motogo" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/mulyawansentosa"><img src="https://avatars1.githubusercontent.com/u/29946673?v=4" alt="mulyawansentosa" title="mulyawansentosa" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/nasoma"><img src="https://avatars1.githubusercontent.com/u/19878418?v=4" alt="nasoma" title="nasoma" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/ozfive"><img src="https://avatars1.githubusercontent.com/u/4494266?v=4" alt="ozfive" title="ozfive" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/paulxu21"><img src="https://avatars1.githubusercontent.com/u/6261758?v=4" alt="paulxu21" title="paulxu21" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/pitt134"><img src="https://avatars1.githubusercontent.com/u/13091629?v=4" alt="pitt134" title="pitt134" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/qiuzhanghua"><img src="https://avatars1.githubusercontent.com/u/478393?v=4" alt="qiuzhanghua" title="qiuzhanghua" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/rapita"><img src="https://avatars1.githubusercontent.com/u/22305375?v=4" alt="rapita" title="rapita" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/remopavithran"><img src="https://avatars1.githubusercontent.com/u/50388068?v=4" alt="remopavithran" title="remopavithran" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/rfunix"><img src="https://avatars1.githubusercontent.com/u/6026357?v=4" alt="rfunix" title="rfunix" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/rhernandez-itemsoft"><img src="https://avatars1.githubusercontent.com/u/4327356?v=4" alt="rhernandez-itemsoft" title="rhernandez-itemsoft" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/risallaw"><img src="https://avatars1.githubusercontent.com/u/15353146?v=4" alt="risallaw" title="risallaw" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/rxrw"><img src="https://avatars1.githubusercontent.com/u/9566402?v=4" alt="rxrw" title="rxrw" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/saleebm"><img src="https://avatars1.githubusercontent.com/u/34875122?v=4" alt="saleebm" title="saleebm" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/sbenimeli"><img src="https://avatars1.githubusercontent.com/u/46652122?v=4" alt="sbenimeli" title="sbenimeli" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/svirmi"><img src="https://avatars1.githubusercontent.com/u/52601346?v=4" alt="svirmi" title="svirmi" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/unixedia"><img src="https://avatars1.githubusercontent.com/u/70646128?v=4" alt="unixedia" title="unixedia" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/vguhesan"><img src="https://avatars1.githubusercontent.com/u/193960?v=4" alt="vguhesan" title="vguhesan" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/vladimir-petukhov-sr"><img src="https://avatars1.githubusercontent.com/u/1183901?v=4" alt="vladimir-petukhov-sr" title="vladimir-petukhov-sr" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/vuhoanglam"><img src="https://avatars1.githubusercontent.com/u/59502855?v=4" alt="vuhoanglam" title="vuhoanglam" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/yonson2"><img src="https://avatars1.githubusercontent.com/u/1192599?v=4" alt="yonson2" title="yonson2" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/SergeShin"><img src="https://avatars1.githubusercontent.com/u/402395?v=4" alt="SergeShin" title="SergeShin" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/blackHoleNgc1277"><img src="https://avatars1.githubusercontent.com/u/41342763?v=4" alt="blackHoleNgc1277" title="blackHoleNgc1277" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/martinlindhe"><img src="https://avatars1.githubusercontent.com/u/181531?v=4" alt="martinlindhe" title="martinlindhe" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/mtrense"><img src="https://avatars1.githubusercontent.com/u/1008285?v=4" alt="mtrense" title="mtrense" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/netbaalzovf"><img src="https://avatars1.githubusercontent.com/u/98529711?v=4" alt="netbaalzovf" title="netbaalzovf" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/lfaynman"><img src="https://avatars1.githubusercontent.com/u/16815068?v=4" alt="lfaynman" title="lfaynman" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/ArturWierzbicki"><img src="https://avatars1.githubusercontent.com/u/23451458?v=4" alt="ArturWierzbicki" title="ArturWierzbicki" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/NA"><img src="https://avatars1.githubusercontent.com/u/1600?v=4" alt="NA" title="NA" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/RainerGevers"><img src="https://avatars1.githubusercontent.com/u/32453861?v=4" alt="RainerGevers" title="RainerGevers" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/aaxx"><img src="https://avatars1.githubusercontent.com/u/476416?v=4" alt="aaxx" title="aaxx" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/crashCoder"><img src="https://avatars1.githubusercontent.com/u/1144298?v=4" alt="crashCoder" title="crashCoder" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/dochoaj"><img src="https://avatars1.githubusercontent.com/u/1789678?v=4" alt="dochoaj" title="dochoaj" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/gog200921"><img src="https://avatars1.githubusercontent.com/u/101519620?v=4" alt="gog200921" title="gog200921" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/nikharsaxena"><img src="https://avatars1.githubusercontent.com/u/8684362?v=4" alt="nikharsaxena" title="nikharsaxena" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/rbondi"><img src="https://avatars1.githubusercontent.com/u/81764?v=4" alt="rbondi" title="rbondi" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/statik"><img src="https://avatars1.githubusercontent.com/u/983?v=4" alt="statik" title="statik" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/thejones"><img src="https://avatars1.githubusercontent.com/u/682850?v=4" alt="thejones" title="thejones" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/vcruzato"><img src="https://avatars1.githubusercontent.com/u/3864151?v=4" alt="vcruzato" title="vcruzato" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/CSRaghunandan"><img src="https://avatars1.githubusercontent.com/u/5226809?v=4" alt="CSRaghunandan" title="CSRaghunandan" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/GeorgeFourikis"><img src="https://avatars1.githubusercontent.com/u/17906313?v=4" alt="GeorgeFourikis" title="GeorgeFourikis" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/L-M-Sherlock"><img src="https://avatars1.githubusercontent.com/u/32575846?v=4" alt="L-M-Sherlock" title="L-M-Sherlock" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/edsongley"><img src="https://avatars1.githubusercontent.com/u/35545454?v=4" alt="edsongley" title="edsongley" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/evan"><img src="https://avatars1.githubusercontent.com/u/210?v=4" alt="evan" title="evan" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/grassshrimp"><img src="https://avatars1.githubusercontent.com/u/3070576?v=4" alt="grassshrimp" title="grassshrimp" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/hazmi-e205"><img src="https://avatars1.githubusercontent.com/u/12555465?v=4" alt="hazmi-e205" title="hazmi-e205" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/jtgoral"><img src="https://avatars1.githubusercontent.com/u/19780595?v=4" alt="jtgoral" title="jtgoral" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/ky2s"><img src="https://avatars1.githubusercontent.com/u/19502125?v=4" alt="ky2s" title="ky2s" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/lauweliam"><img src="https://avatars1.githubusercontent.com/u/4064517?v=4" alt="lauweliam" title="lauweliam" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/letmestudy"><img src="https://avatars1.githubusercontent.com/u/31943708?v=4" alt="letmestudy" title="letmestudy" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/mblandr"><img src="https://avatars1.githubusercontent.com/u/42862020?v=4" alt="mblandr" title="mblandr" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/ndimorle"><img src="https://avatars1.githubusercontent.com/u/76732415?v=4" alt="ndimorle" title="ndimorle" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/primadi"><img src="https://avatars1.githubusercontent.com/u/7625413?v=4" alt="primadi" title="primadi" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/shyyawn"><img src="https://avatars1.githubusercontent.com/u/6064438?v=4" alt="shyyawn" title="shyyawn" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/wangbl11"><img src="https://avatars1.githubusercontent.com/u/14358532?v=4" alt="wangbl11" title="wangbl11" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/wofka72"><img src="https://avatars1.githubusercontent.com/u/10855340?v=4" alt="wofka72" title="wofka72" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/xsokev"><img src="https://avatars1.githubusercontent.com/u/28113?v=4" alt="xsokev" title="xsokev" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/oleang"><img src="https://avatars1.githubusercontent.com/u/142615?v=4" alt="oleang" title="oleang" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/michalsz"><img src="https://avatars1.githubusercontent.com/u/187477?v=4" alt="michalsz" title="michalsz" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/Curtman"><img src="https://avatars1.githubusercontent.com/u/543481?v=4" alt="Curtman" title="Curtman" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/claudemuller"><img src="https://avatars1.githubusercontent.com/u/8104894?v=4" alt="claudemuller" title="claudemuller" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/SridarDhandapani"><img src="https://avatars1.githubusercontent.com/u/18103118?v=4" alt="SridarDhandapani" title="SridarDhandapani" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/midhubalan"><img src="https://avatars1.githubusercontent.com/u/13059634?v=4" alt="midhubalan" title="midhubalan" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/rosales-stephanie"><img src="https://avatars1.githubusercontent.com/u/43592017?v=4" alt="rosales-stephanie" title="rosales-stephanie" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/opusmagna"><img src="https://avatars1.githubusercontent.com/u/33766678?v=4" alt="opusmagna" title="opusmagna" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/b4zz4r"><img src="https://avatars1.githubusercontent.com/u/7438782?v=4" alt="b4zz4r" title="b4zz4r" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/bobmcallan"><img src="https://avatars1.githubusercontent.com/u/8773580?v=4" alt="bobmcallan" title="bobmcallan" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/fangli"><img src="https://avatars1.githubusercontent.com/u/3032639?v=4" alt="fangli" title="fangli" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/galois-tnp"><img src="https://avatars1.githubusercontent.com/u/41128011?v=4" alt="galois-tnp" title="galois-tnp" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/geoshan"><img src="https://avatars1.githubusercontent.com/u/10161131?v=4" alt="geoshan" title="geoshan" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/juanxme"><img src="https://avatars1.githubusercontent.com/u/661043?v=4" alt="juanxme" title="juanxme" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/nguyentamvinhlong"><img src="https://avatars1.githubusercontent.com/u/1875916?v=4" alt="nguyentamvinhlong" title="nguyentamvinhlong" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/tejzpr"><img src="https://avatars1.githubusercontent.com/u/2813811?v=4" alt="tejzpr" title="tejzpr" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/theantichris"><img src="https://avatars1.githubusercontent.com/u/1486502?v=4" alt="theantichris" title="theantichris" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/tuxaanand"><img src="https://avatars1.githubusercontent.com/u/9750371?v=4" alt="tuxaanand" title="tuxaanand" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/narven"><img src="https://avatars1.githubusercontent.com/u/123594?v=4" alt="narven" title="narven" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/raphael-brand"><img src="https://avatars1.githubusercontent.com/u/4279168?v=4" alt="raphael-brand" title="raphael-brand" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/Tang634724712"><img src="https://avatars1.githubusercontent.com/u/30973119?v=4" alt="Tang634724712" title="Tang634724712" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/HieuLsw"><img src="https://avatars1.githubusercontent.com/u/1675478?v=4" alt="HieuLsw" title="HieuLsw" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/carlosmoran092"><img src="https://avatars1.githubusercontent.com/u/10361754?v=4" alt="carlosmoran092" title="carlosmoran092" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
  <a href="https://github.com/yangxianglong"><img src="https://avatars1.githubusercontent.com/u/55280276?v=4" alt="yangxianglong" title="yangxianglong" with="75" style="width:75px;max-width:75px;height:75px" height="75" /></a>
</p>

## 📖 Learning Iris

### Create a new project

```sh
$ mkdir myapp
$ cd myapp
$ go mod init myapp
$ go get github.com/kataras/iris/v12@master # or @v12.2.0-beta2
```

<details><summary>Install on existing project</summary>

```sh
$ cd myapp
$ go get github.com/kataras/iris/v12@master
```

**Run**

```sh
$ go mod tidy -compat=1.18
$ go run .
```

</details>

![](https://www.iris-go.com/images/gifs/install-create-iris.gif)

Iris contains extensive and thorough **[documentation](https://www.iris-go.com/docs)** making it easy to get started with the framework.

<!-- Iris contains extensive and thorough **[wiki](https://github.com/kataras/iris/wiki)** making it easy to get started with the framework. -->

<!-- ![](https://media.giphy.com/media/Ur8iqy9FQfmPuyQpgy/giphy.gif) -->

For a more detailed technical documentation you can head over to our [godocs](https://pkg.go.dev/github.com/kataras/iris/v12@master). And for executable code you can always visit the [./_examples](_examples) repository's subdirectory.

### Do you like to read while traveling?

<a href="https://iris-go.com/#book"> <img alt="Book cover" src="https://iris-go.com/images/iris-book-cover-sm.jpg?v=12" /> </a>

[![follow author on twitter](https://img.shields.io/twitter/follow/makismaropoulos?color=3D8AA3&logoColor=3D8AA3&style=for-the-badge&logo=twitter)](https://twitter.com/intent/follow?screen_name=makismaropoulos)

[![follow Iris web framework on twitter](https://img.shields.io/twitter/follow/iris_framework?color=ee7506&logoColor=ee7506&style=for-the-badge&logo=twitter)](https://twitter.com/intent/follow?screen_name=iris_framework)

[![follow Iris web framework on facebook](https://img.shields.io/badge/Follow%20%40Iris.framework-522-2D88FF.svg?style=for-the-badge&logo=facebook)](https://www.facebook.com/iris.framework)

You can [request](https://www.iris-go.com/#ebookDonateForm) a PDF and online access of the **Iris E-Book** (New Edition, **future v12.2.0+**) today and be participated in the development of Iris.

## 🙌 Contributing

We'd love to see your contribution to the Iris Web Framework! For more information about contributing to the Iris project please check the [CONTRIBUTING.md](CONTRIBUTING.md) file.

[List of all Contributors](https://github.com/kataras/iris/graphs/contributors)

## 🛡 Security Vulnerabilities

If you discover a security vulnerability within Iris, please send an e-mail to [iris-go@outlook.com](mailto:iris-go@outlook.com). All security vulnerabilities will be promptly addressed.

## 📝 License

This project is licensed under the [BSD 3-clause license](LICENSE), just like the Go project itself.

The project name "Iris" was inspired by the Greek mythology.
<!-- ## Stargazers over time

[![Stargazers over time](https://starchart.cc/kataras/iris.svg)](https://starchart.cc/kataras/iris) -->
