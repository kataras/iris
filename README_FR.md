# <a href="https://iris-go.com"><img src="https://iris-go.com/images/logo-new-lq-45.png"></a> Iris Web Framework <a href="README_GR.md"><img width="20px" src="https://iris-go.com/images/flag-greece.svg" /> <a href="README_JA.md"><img width="20px" height="20px" src="https://iris-go.com/images/flag-japan.svg" /></a> </a> <a href="README_FR.md"><img width="20px" src="https://iris-go.com/images/flag-france.svg" /></a> <a href="README_ZH_HANT.md"><img width="20px" src="https://iris-go.com/images/flag-taiwan.svg" /></a> <a href="README_ZH_HANS.md"><img width="20px" src="https://iris-go.com/images/flag-china.svg" /></a> <a href="README_ES.md"><img width="20px" src="https://iris-go.com/images/flag-spain.png" /></a> <a href="README_FA.md"><img width="20px" src="https://iris-go.com/images/flag-iran.svg" /></a> <a href="README_RU.md"><img width="20px" src="https://iris-go.com/images/flag-russia.svg" /></a> <a href="README_KO.md"><img width="20px" src="https://iris-go.com/images/flag-south-korea.svg?v=12" /></a> <a href="README_PT_BR.md"><img width="20px" height="20px" src="https://iris-go.com/images/flag-brazil.svg" /></a> <a href="README_VN.md"><img width="20px" height="20px" src="https://iris-go.com/images/flag-vietnam.svg" /></a>

[![build status](https://img.shields.io/github/actions/workflow/status/kataras/iris/ci.yml?branch=main&style=for-the-badge)](https://github.com/kataras/iris/actions/workflows/ci.yml) [![release](https://img.shields.io/badge/release%20-v14.0-0077b3.svg?style=for-the-badge)](https://github.com/kataras/iris/releases) [![view examples](https://img.shields.io/badge/examples%20-59-a83adf.svg?style=for-the-badge&logo=go)](_examples/README.md) [![godocs](https://img.shields.io/badge/go-%20docs-488AC7.svg?style=for-the-badge)](https://pkg.go.dev/github.com/kataras/iris/v14@v14.0.0) [![chat](https://img.shields.io/gitter/room/iris_go/community.svg?color=cc2b5e&logo=gitter&style=for-the-badge)](https://gitter.im/iris_go/community) [![donate](https://img.shields.io/badge/support-Iris-blue.svg?style=for-the-badge&logo=paypal)](https://iris-go.com/donate)

Iris est un framework web pour Go, rapide et livré avec tout le nécessaire. Il apporte à votre prochaine API ou site web un routage expressif avec des paramètres de chemin typés, un point unique pour la gestion des erreurs, une injection de dépendances qui garde le code testable et des réglages de production sûrs par défaut.

La **version 14** est la plus grande publication de l'histoire du projet. Le framework a été reconstruit autour des génériques de Go : un builder d'application vérifié à la compilation, des helpers de requête typés qui valident pour vous, des contrôleurs résolus par le conteneur de dépendances et un pipeline d'erreurs que vous configurez une seule fois. [HISTORY.md](HISTORY.md) recense toutes les nouveautés ; le [Guide de Migration](MIGRATION.md) fait passer les applications v12 à la nouvelle version en une seule séance.

> 🤖 **Interrogez l'IA sur Iris.** Le [Code Wiki d'Iris](https://codewiki.google/github.com/kataras/iris), gratuit et proposé par Google, est généré à partir de ce dépôt et reste synchronisé avec lui, commit après commit. Parcourez l'architecture avec des diagrammes, ou posez à son chat des questions en langage courant sur le code et obtenez des réponses fondées sur la source réelle.

Découvrez ce que [d'autres disent d'Iris](https://www.iris-go.com/#review) et mettez une **[étoile](https://github.com/kataras/iris/stargazers)** à ce projet open source pour soutenir son développement. La vitesse vous intéresse ? Consultez les [benchmarks de serveurs](https://github.com/kataras/server-benchmarks) indépendants.

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

<details><summary>La pile de production, en une seule expression</summary>

Les applications réelles utilisent `iris.NewBuilder` : une chaîne vérifiée à la compilation qui câble CORS, la compression, les logs d'accès, un endpoint de santé, votre table d'erreurs, vos services et vos groupes d'API. Le compilateur rejette un ordre d'étapes incorrect.

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

Les handlers tiennent en six lignes, parce que le décodage, la validation et le rendu des erreurs se font en un seul endroit configuré :

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

Le [Tutoriel de Démarrage](TUTORIAL.md) construit un service complet et testé sur cette pile, dans un seul document.

</details>

Comme l'a dit un jour un [développeur Go](https://twitter.com/dkuye/status/1532087942696554497), **Iris vous couvre sur tous les fronts et tient bon au fil des années**.

Ce que vous obtenez avec Iris v14 :

* Routage avec paramètres de chemin typés (`{id:uuid}`, `{n:int}`, `{addr:email}` et plus), macros personnalisées et contraintes. Les entrées invalides sont rejetées avant l'exécution de votre handler.
* `iris.NewBuilder` : CORS, compression, journalisation des accès, endpoint de santé, mappage des erreurs, services et groupes d'API assemblés dans une chaîne vérifiée à la compilation.
* Injection de dépendances : les constructeurs déclarent ce dont ils ont besoin, le conteneur le fournit. Hooks de démarrage `Initializer`, closers d'arrêt automatiques.
* Les helpers `rest` et une table d'erreurs centrale : des handlers minces, des erreurs canoniques lisibles par machine, les détails internes ne fuient jamais vers les clients.
* Des payloads de requête qui se valident eux-mêmes : implémentez `Validate() error` et `ctx.BindJSON` l'exécute après le décodage.
* Trente et un middlewares intégrés : accesslog, API keys, basicauth, bodylimit, cache, CORS, hCaptcha et reCAPTCHA, JWT, monitor, pprof, rate limiting, recovery, request IDs, rewrite, sessions, versioning et plus.
* L'authentification à toutes les échelles : Basic Auth en une ligne, clés d'API, un SDK JWT générique, un serveur d'identité complet et un fournisseur OAuth2.
* Des contrôleurs MVC bâtis sur les génériques de Go.
* WebSockets, service gRPC et versionnement d'API.
* Moteurs de vues : HTML, Django, Handlebars, Pug et plus via iris-contrib/views.
* Localisation avec pluralisation et détection de langue, plus les sitemaps.
* Des sessions avec des cookies sûrs par défaut (`Secure`, `SameSite=Lax` d'origine).
* Des réponses riches : JSON, JSONP, XML, YAML, Markdown, Protocol Buffers, MessagePack, négociation de contenu, streaming et Server-Sent Events.
* Backends d'IA : le package `ai` assure le rendu et le relais via HTTP de la sortie en streaming des modèles (deltas de texte, traces de raisonnement, fragments d'appels d'outils, usage). Il ne dépend d'aucun fournisseur et n'embarque aucun de leurs SDK.
* Un client HTTP, le package `client`, pour appeler d'autres services avec les mêmes règles JSON que le côté serveur.
* Compression des réponses : gzip, deflate, brotli, snappy, s2, zstd.
* TLS avec Let's Encrypt, et un tunnel ngrok intégré (opt-in: `import _ "github.com/kataras/iris/v14/core/server/ngrok"`) pour mettre une application de dev sur internet en quelques secondes.
* Tests HTTP en mémoire : des tests de l'application complète sans ports, sans réseau et sans sleeps.
* Un livre complet à lire en ligne et à télécharger gratuitement, un tutoriel pratique et un wiki IA en direct (voir Apprendre Iris ci-dessous).
* Et le plus important : vous obtenez des réponses rapides et du soutien, du premier jour jusqu'à aujourd'hui.

## 📖 Apprendre Iris

> Nouveau sur Iris ? Commencez par le [Tutoriel de Démarrage](TUTORIAL.md) : un seul document qui vous mène de `go mod init` à une API REST testée, prête pour la production. Vous migrez depuis la v12 ? Consultez le [Guide de Migration](MIGRATION.md). Une question en cours de route ? Interrogez le **wiki IA en direct**, gratuit, sur [codewiki.google/github.com/kataras/iris](https://codewiki.google/github.com/kataras/iris), généré par Google à partir de la source d'Iris et maintenu en synchronisation avec elle, avec un chat intégré qui répond aux questions sur le code.

### Installation

Le seul prérequis est le [langage de programmation Go](https://go.dev/dl/), version 1.27 ou ultérieure.

#### Créer un nouveau projet

```sh
$ mkdir myapp
$ cd myapp
$ go mod init myapp
$ go get github.com/kataras/iris/v14@latest # or @v14.0.0
```

<details><summary>Installer sur un projet existant</summary>

```sh
$ cd myapp
$ go get github.com/kataras/iris/v14@latest
```

**Exécuter**

```sh
$ go mod tidy
$ go run .
```

</details>

![](https://www.iris-go.com/images/gifs/install-create-iris.gif)

Iris propose une **[documentation](https://www.iris-go.com/docs)** étendue et détaillée qui facilite la prise en main du framework.

Pour une documentation technique plus détaillée, rendez-vous sur nos [godocs](https://pkg.go.dev/github.com/kataras/iris/v14@v14.0.0). Pour une introduction complète et pratique à la pile recommandée de la v14, lisez le [Tutoriel de Démarrage](TUTORIAL.md), et si vous migrez depuis la v12, suivez le [Guide de Migration](MIGRATION.md).

Vous préférez apprendre par le code ? Les [**59 exemples**](_examples/README.md) vont d'une première route à l'injection de dépendances, l'authentification, les tests et les points d'extension du framework. Chacun d'eux a été compilé et éprouvé via HTTP avant d'être commité.

### Le Livre d'Iris

<a href="https://iris-go.com/book/"> <img alt="Book cover" src="https://iris-go.com/images/iris-book-cover-sm.jpg?v=14" /> </a>

Le livre complet comprend une préface, 24 chapitres et un épilogue, chaque exemple de code vérifié contre la source du framework. Chaque chapitre s'ouvre sur une mise en contexte en langage simple pour les débutants et se termine par des lectures complémentaires choisies. Chaque chapitre est en lecture libre sur [iris-go.com/docs](https://iris-go.com/docs/) ; la [page du livre](https://iris-go.com/book/) donne le sommaire complet.

Le PDF est gratuit lui aussi. [Téléchargez-le directement](https://iris-go.com/iris-book.pdf) : il n'y a aucun formulaire à remplir et rien à demander. Si le livre vous est utile, [sponsoriser le projet](https://iris-go.com/donate/) est la façon de prendre part au développement d'Iris.

[![follow author on twitter](https://img.shields.io/twitter/follow/makismaropoulos?color=3D8AA3&logoColor=3D8AA3&style=for-the-badge&logo=twitter)](https://twitter.com/intent/follow?screen_name=makismaropoulos)

[![follow Iris web framework on twitter](https://img.shields.io/twitter/follow/iris_framework?color=ee7506&logoColor=ee7506&style=for-the-badge&logo=twitter)](https://twitter.com/intent/follow?screen_name=iris_framework)

[![follow Iris web framework on facebook](https://img.shields.io/badge/Follow%20%40Iris.framework-569-2D88FF.svg?style=for-the-badge&logo=facebook)](https://www.facebook.com/iris.framework)

## 🤖 Outils d'IA pour Iris

Deux outils d'IA connaissent ce framework en particulier, au-delà de ce qu'un modèle généraliste sait de Go.

Le [wiki IA en direct](https://codewiki.google/github.com/kataras/iris) est gratuit et fonctionne dans votre navigateur : parcourez l'architecture ou interrogez son chat sur le code.

[Plexon AI](https://plexon.ai) est un assistant de bureau pour Windows, macOS et Linux, créé par l'auteur d'Iris. Il intègre la skill Iris ([iris-go.com/ai](https://iris-go.com/ai/)) : l'API v14, la chaîne du builder, les helpers `rest`, le mappage des erreurs, le SDK d'authentification et les modèles de tests, dans vingt et un documents de référence que l'assistant charge à la demande pendant qu'il écrit votre code. Activez la skill depuis le panneau Skills, ou installez la [persona Software Developer](https://plexon.ai/personas/software-developer/), qui l'active avec le reste de son outillage. [Téléchargez Plexon AI](https://plexon.ai/download/).

Pour les sponsors, la contribution, la politique de sécurité et la licence, consultez le [README anglais](README.md).

Ce projet est sous [licence BSD 3-clause](LICENSE), tout comme le projet Go lui-même.
