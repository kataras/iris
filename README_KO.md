# <a href="https://iris-go.com"><img src="https://iris-go.com/images/logo-new-lq-45.png"></a> Iris Web Framework <a href="README_GR.md"><img width="20px" src="https://iris-go.com/images/flag-greece.svg" /> <a href="README_JA.md"><img width="20px" height="20px" src="https://iris-go.com/images/flag-japan.svg" /></a> </a> <a href="README_FR.md"><img width="20px" src="https://iris-go.com/images/flag-france.svg" /></a> <a href="README_ZH_HANT.md"><img width="20px" src="https://iris-go.com/images/flag-taiwan.svg" /></a> <a href="README_ZH_HANS.md"><img width="20px" src="https://iris-go.com/images/flag-china.svg" /></a> <a href="README_ES.md"><img width="20px" src="https://iris-go.com/images/flag-spain.png" /></a> <a href="README_FA.md"><img width="20px" src="https://iris-go.com/images/flag-iran.svg" /></a> <a href="README_RU.md"><img width="20px" src="https://iris-go.com/images/flag-russia.svg" /></a> <a href="README_KO.md"><img width="20px" src="https://iris-go.com/images/flag-south-korea.svg?v=12" /></a> <a href="README_PT_BR.md"><img width="20px" height="20px" src="https://iris-go.com/images/flag-brazil.svg" /></a> <a href="README_VN.md"><img width="20px" height="20px" src="https://iris-go.com/images/flag-vietnam.svg" /></a>

[![build status](https://img.shields.io/github/actions/workflow/status/kataras/iris/ci.yml?branch=main&style=for-the-badge)](https://github.com/kataras/iris/actions/workflows/ci.yml) [![release](https://img.shields.io/badge/release%20-v14.0-0077b3.svg?style=for-the-badge)](https://github.com/kataras/iris/releases) [![view examples](https://img.shields.io/badge/examples%20-59-a83adf.svg?style=for-the-badge&logo=go)](_examples/README.md) [![godocs](https://img.shields.io/badge/go-%20docs-488AC7.svg?style=for-the-badge)](https://pkg.go.dev/github.com/kataras/iris/v14@v14.0.0) [![chat](https://img.shields.io/gitter/room/iris_go/community.svg?color=cc2b5e&logo=gitter&style=for-the-badge)](https://gitter.im/iris_go/community) [![donate](https://img.shields.io/badge/support-Iris-blue.svg?style=for-the-badge&logo=paypal)](https://iris-go.com/donate)

Iris는 Go를 위한 빠르고 필요한 것이 모두 갖춰진(batteries-included) 웹 프레임워크입니다. 여러분의 다음 API나 웹사이트에 타입 지정 경로 파라미터를 갖춘 표현력 있는 라우팅, 한곳에서 처리하는 에러 핸들링, 코드를 테스트 가능하게 유지해 주는 의존성 주입, 그리고 기본부터 안전한 프로덕션 기본값을 제공합니다.

**버전 14**는 프로젝트 역사상 가장 큰 릴리스입니다. 프레임워크는 Go 제네릭을 중심으로 재구축되었습니다: 컴파일 타임에 강제되는 애플리케이션 빌더, 검증을 대신해 주는 타입 지정 요청 헬퍼, 의존성 컨테이너가 해결하는 컨트롤러, 그리고 한 번만 구성하면 되는 에러 파이프라인. 새로운 내용은 모두 [HISTORY.md](HISTORY.md)에 기록되어 있으며, [마이그레이션 가이드](MIGRATION.md)를 따르면 v12 애플리케이션을 한 번에 최신으로 끌어올릴 수 있습니다.

> 🤖 **AI에게 Iris에 대해 물어보세요.** Google이 무료로 제공하는 [Iris Code Wiki](https://codewiki.google/github.com/kataras/iris)는 이 저장소로부터 생성되며, 커밋 하나하나에 맞춰 동기화 상태를 유지합니다. 다이어그램과 함께 아키텍처를 둘러보거나, 채팅에 코드베이스에 대해 일상 언어로 질문하고 실제 소스에 근거한 답변을 받아 보세요.

[다른 사람들이 Iris에 대해 말하는 것](https://www.iris-go.com/#review)을 확인하고, 이 오픈 소스 프로젝트의 개발을 지원하기 위해 **[스타](https://github.com/kataras/iris/stargazers)**를 눌러 주세요. 속도가 궁금하신가요? 독립적으로 수행된 [서버 벤치마크](https://github.com/kataras/server-benchmarks)를 확인해 보세요.

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

<details><summary>프로덕션 스택, 하나의 표현식으로</summary>

실제 애플리케이션은 `iris.NewBuilder`를 사용합니다. CORS, 압축, 액세스
로그, 헬스 엔드포인트, 에러 맵, 서비스, API 그룹을 하나로 연결하는,
컴파일 타임에 강제되는 체인입니다. 단계 순서가 잘못되면 컴파일러가
거부합니다.

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

디코딩, 검증, 에러 렌더링이 미리 구성된 한곳에서 이루어지기 때문에
핸들러는 여섯 줄로 유지됩니다:

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

[시작하기 튜토리얼](TUTORIAL.md)은 이 스택 위에 완전하고 테스트까지 갖춘
서비스를 문서 하나로 구축합니다.

</details>


한 [Go 개발자](https://twitter.com/dkuye/status/1532087942696554497)가 말했듯이, **Iris는 모든 면에서 여러분을 든든하게 지원하며 수년간 굳건히 자리를 지켜 왔습니다**.

Iris v14가 제공하는 것:

* 타입 지정 경로 파라미터(`{id:uuid}`, `{n:int}`, `{addr:email}` 등), 커스텀 매크로와 제약 조건을 갖춘 라우팅. 잘못된 입력은 핸들러가 실행되기 전에 거부됩니다.
* `iris.NewBuilder`: CORS, 압축, 액세스 로깅, 헬스 엔드포인트, 에러 매핑, 서비스, API 그룹을 컴파일 타임에 검사되는 하나의 체인으로 조립합니다.
* 의존성 주입: 생성자가 필요한 것을 선언하면 컨테이너가 제공합니다. 시작 시 `Initializer` 훅, 종료 시 자동 리소스 정리.
* `rest` 헬퍼와 하나의 중앙 에러 맵: 얇은 핸들러, 표준화된 기계 판독 가능 에러, 내부 정보는 클라이언트로 절대 새어 나가지 않습니다.
* 스스로 검증하는 요청 페이로드: `Validate() error`를 구현하면 `ctx.BindJSON`이 디코딩 후 실행합니다.
* 31개의 내장 미들웨어: accesslog, API 키, basicauth, bodylimit, cache, CORS, hCaptcha와 reCAPTCHA, JWT, monitor, pprof, 속도 제한(rate limiting), recovery, 요청 ID, rewrite, 세션, 버저닝 등.
* 모든 규모의 인증: 한 줄로 끝나는 Basic Auth, API 키, 제네릭 JWT SDK, 완전한 아이덴티티 서버와 OAuth2 프로바이더.
* Go 제네릭 기반으로 구축된 MVC 컨트롤러.
* WebSocket, gRPC 서빙, API 버저닝.
* 뷰 엔진: HTML, Django, Handlebars, Pug 등(iris-contrib/views를 통해).
* 복수형 처리와 언어 감지를 갖춘 로컬라이제이션, 그리고 사이트맵.
* 안전한 쿠키 기본값을 갖춘 세션(`Secure`, `SameSite=Lax` 기본 적용).
* 풍부한 응답: JSON, JSONP, XML, YAML, Markdown, Protocol Buffers, MessagePack, 콘텐츠 협상, 스트리밍, Server-Sent Events.
* AI 백엔드: `ai` 패키지는 스트리밍되는 모델 출력(텍스트 델타, 추론 트레이스, 도구 호출 조각, 사용량)을 HTTP로 렌더링하고 중계합니다. 특정 프로바이더에 종속되지 않으며 벤더 SDK를 포함하지 않습니다.
* 서버 측과 같은 JSON 규칙으로 다른 서비스를 호출하는 HTTP 클라이언트, `client` 패키지.
* 응답 압축: gzip, deflate, brotli, snappy, s2, zstd.
* Let's Encrypt를 통한 TLS, 그리고 개발 중인 앱을 몇 초 만에 인터넷에 공개하는 내장 ngrok 터널링 (opt-in: `import _ "github.com/kataras/iris/v14/core/server/ngrok"`).
* 인메모리 HTTP 테스트: 포트도, 네트워크도, sleep도 없는 전체 애플리케이션 테스트.
* 온라인에서 무료로 읽고 내려받을 수 있는 완전한 책, 실습형 튜토리얼, 라이브 AI 위키(아래 'Iris 배우기' 참고).
* 그리고 가장 중요한 것: 첫날부터 지금까지, 빠른 답변과 지원을 받을 수 있습니다.

## 📖 Iris 배우기

> Iris가 처음이신가요? [시작하기 튜토리얼](TUTORIAL.md)부터 시작하세요.
> `go mod init`에서 테스트를 갖춘 프로덕션 형태의 REST API까지 문서
> 하나로 안내합니다. v12에서 업그레이드하시나요?
> [마이그레이션 가이드](MIGRATION.md)를 참고하세요. 진행 중에 궁금한 점이
> 있다면 무료 **라이브 AI 위키**
> [codewiki.google/github.com/kataras/iris](https://codewiki.google/github.com/kataras/iris)
> 에 물어보세요. Google이 Iris 소스로부터 생성해 항상 동기화 상태를
> 유지하며, 내장 채팅이 코드베이스에 대한 질문에 답해 줍니다.

### 설치

유일한 요구 사항은 [Go 프로그래밍 언어](https://go.dev/dl/) 1.27 이상 버전입니다.

#### 새 프로젝트 만들기

```sh
$ mkdir myapp
$ cd myapp
$ go mod init myapp
$ go get github.com/kataras/iris/v14@latest # or @v14.0.0
```

<details><summary>기존 프로젝트에 설치</summary>

```sh
$ cd myapp
$ go get github.com/kataras/iris/v14@latest
```

**실행**

```sh
$ go mod tidy
$ go run .
```

</details>

![](https://www.iris-go.com/images/gifs/install-create-iris.gif)

Iris는 방대하고 꼼꼼한 **[문서](https://www.iris-go.com/docs)**를 갖추고 있어 프레임워크를 쉽게 시작할 수 있습니다.

더 자세한 기술 문서는 [godocs](https://pkg.go.dev/github.com/kataras/iris/v14@v14.0.0)를 참고하세요. 권장 v14 스택에 대한 완전한 실습형 입문으로는 [시작하기 튜토리얼](TUTORIAL.md)을 읽고, v12에서 업그레이드한다면 [마이그레이션 가이드](MIGRATION.md)를 따라가세요.

코드로 배우는 편이 좋다면, [**58개의 예제**](_examples/README.md)가 첫 라우트부터 의존성 주입, 인증, 테스트, 프레임워크의 확장 지점까지 이어집니다. 모두 커밋되기 전에 컴파일하고 HTTP로 직접 동작을 확인했습니다.

### Iris 북

<a href="https://iris-go.com/book/"> <img alt="Book cover" src="https://iris-go.com/images/iris-book-cover-sm.jpg?v=14" /> </a>

완전한 책은 서문, 24개의 장, 에필로그로 구성되며, 모든 코드 예제는 프레임워크 소스와 대조해 검증되었습니다. 각 장은 입문자를 위한 쉬운 배경 설명으로 시작해 엄선된 추가 읽을거리로 마무리됩니다. 모든 장은 [iris-go.com/docs](https://iris-go.com/docs/)에서 무료로 읽을 수 있으며, 전체 목차는 [책 페이지](https://iris-go.com/book/)에 있습니다.

PDF도 무료입니다. [바로 내려받으세요](https://iris-go.com/iris-book.pdf). 작성할 양식도, 요청할 것도 없습니다. 책이 도움이 되었다면 [프로젝트를 후원](https://iris-go.com/donate/)하는 것이 Iris 개발에 참여하는 방법입니다.

[![follow author on twitter](https://img.shields.io/twitter/follow/makismaropoulos?color=3D8AA3&logoColor=3D8AA3&style=for-the-badge&logo=twitter)](https://twitter.com/intent/follow?screen_name=makismaropoulos)

[![follow Iris web framework on twitter](https://img.shields.io/twitter/follow/iris_framework?color=ee7506&logoColor=ee7506&style=for-the-badge&logo=twitter)](https://twitter.com/intent/follow?screen_name=iris_framework)

[![follow Iris web framework on facebook](https://img.shields.io/badge/Follow%20%40Iris.framework-569-2D88FF.svg?style=for-the-badge&logo=facebook)](https://www.facebook.com/iris.framework)

## 🤖 Iris를 위한 AI 도구

일반 모델이 Go에 대해 아는 수준을 넘어, 이 framework를 구체적으로 이해하는 AI 도구가 두 가지 있습니다.

[라이브 AI wiki](https://codewiki.google/github.com/kataras/iris)는 무료이며 브라우저에서 실행됩니다. 아키텍처를 살펴보거나 채팅으로 코드베이스에 대해 질문할 수 있습니다.

[Plexon AI](https://plexon.ai)는 Windows, macOS, Linux용 데스크톱 어시스턴트로, Iris 작성자가 만들었습니다. Iris skill([iris-go.com/ai](https://iris-go.com/ai/))을 내장하고 있습니다. v14 API, builder 체인, `rest` 헬퍼, 오류 매핑, 인증 SDK, 테스트 패턴이 21개의 참조 문서로 들어 있으며, 어시스턴트가 코드를 작성하면서 필요한 문서를 그때그때 읽습니다. Skills 패널에서 skill을 켜거나, [Software Developer 페르소나](https://plexon.ai/personas/software-developer/)를 설치하면 나머지 개발 도구와 함께 켜집니다. [Plexon AI 내려받기](https://plexon.ai/download/).

---

서포터, 기여, 보안 정책, 라이선스에 대해서는 [영문 README](README.md)를 참고하세요.

이 프로젝트는 Go 프로젝트와 마찬가지로 [BSD 3-clause 라이선스](LICENSE)에 따라 라이선스가 부여됩니다.
