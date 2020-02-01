# Iris <a href="README.md"> <img width="20px" src="https://iris-go.com/images/flag-unitedkingdom.svg?v=10" /></a> <a href="README_ZH.md"><img width="20px" src="https://iris-go.com/images/flag-china.svg?v=10" /></a> <a href="README_GR.md"><img width="20px" src="https://iris-go.com/images/flag-greece.svg?v=10" /></a> <a href="README_ES.md"><img width="20px" src="https://iris-go.com/images/flag-spain.png" /></a> <a href="README_FA.md"><img width="20px" src="https://iris-go.com/images/flag-iran.svg" /></a> <a href="README_RU.md"><img width="20px" src="https://iris-go.com/images/flag-russia.svg" /></a>

[![build status](https://img.shields.io/travis/kataras/iris/master.svg?style=for-the-badge&logo=travis)](https://travis-ci.org/kataras/iris) [![report card](https://img.shields.io/badge/report%20card-a%2B-ff3333.svg?style=for-the-badge)](https://goreportcard.com/report/github.com/kataras/iris)<!--[![godocs](https://img.shields.io/badge/go-%20docs-488AC7.svg?style=for-the-badge)](https://godoc.org/github.com/kataras/iris)--> [![view examples](https://img.shields.io/badge/learn%20by-examples-0077b3.svg?style=for-the-badge)](https://github.com/kataras/iris/tree/master/_examples) [![chat](https://img.shields.io/gitter/room/iris_go/community.svg?color=blue&logo=gitter&style=for-the-badge)](https://gitter.im/iris_go/community) [![release](https://img.shields.io/badge/release%20-v11.2-0077b3.svg?style=for-the-badge)](https://github.com/kataras/iris/releases)

Iris는 단순하고 빠르며 좋은 성능과 모든 기능을 갖춘 Go언어용 웹 프레임워크입니다. 당신의 웹사이트나 API를 위해서 아름답고 사용하기 쉬운 기반을 제공합니다.

[여러 사람들의 의견](https://iris-go.com/testimonials/)을 둘러보세요. 그리고 이 github repository을 **star**하세요.

[![](https://media.giphy.com/media/j5WLmtvwn98VPrm7li/giphy.gif)](https://iris-go.com/testimonials/)

## Iris 배우기

<details>
<summary>일단 해보기</summary>

```sh
# 다음 코드를 example.go 화일에 입력하세요.
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

    app.Run(iris.Addr(":8080"))
}
```

```sh
# example.go 를 실행하고,
# 웹브라우저에서 http://localhost:8080/ping 를 열어보세요.
$ go run example.go
```

> 라우팅은 Go로 작성한 가장 강력하고 빠른 trie기반의 소프트웨어인 [muxie](https://github.com/kataras/muxie)로 처리합니다.

</details>

Iris는 광범위하고 꼼꼼한 **[wiki](https://github.com/kataras/iris/wiki)** 를 가지고 있기 때문에 쉽게 프레임워크를 시작할 수 있습니다.

더 자세한 기술문서를 보시려면 [godocs](https://godoc.org/github.com/kataras/iris)를 방문하세요. 그리고 실행가능한 예제코드는 [\_examples](_examples/) 하위 디렉토리에 있습니다.

### 여행하면서 독서를 즐기세요?

<a href="https://bit.ly/iris-req-book"> <img alt="Book cover" src="https://iris-go.com/images/iris-book-cover-sm.jpg" width="200" /> </a>

[![follow author](https://img.shields.io/twitter/follow/makismaropoulos.svg?style=for-the-badge)](https://twitter.com/intent/follow?screen_name=makismaropoulos)

PDF 버전과 **E-Book** 에 대한 온라인 접근을 [요청](https://bit.ly/iris-req-book)하시고 Iris의 개발에 참가하실 수 있습니다.

## 기여하기

Iris 웹 프레임워크에 대한 여러분의 기여를 환영합니다! Iris 프로젝트에 기여하는 방법에 대한 자세한 내용은 [CONTRIBUTING.md](CONTRIBUTING.md) 파일을 참조하십시오.

[기여자 리스트](https://github.com/kataras/iris/graphs/contributors)

## 보안 취약점

만약 Iris에서 보안 취약점을 발견하시면 [iris-go@outlook.com](mailto:iris-go@outlook.com) 로 메일을 보내주세요. 모든 보안 취약점은 즉 해결할 것입니다.

## 라이센스

이 프로젝트의 이름 "Iris"는 그리스 신화에서 영감을 받았습니다.

Iris 웹 프레임워크는 [3-Clause BSD License](LICENSE)를 가지는 무료 오픈소스 소프트웨어입니다.
