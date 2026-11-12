# <a href="https://iris-go.com"><img src="https://iris-go.com/images/logo-new-lq-45.png"></a> Iris Web Framework <a href="README_GR.md"><img width="20px" src="https://iris-go.com/images/flag-greece.svg" /> <a href="README_JA.md"><img width="20px" height="20px" src="https://iris-go.com/images/flag-japan.svg" /></a> </a> <a href="README_FR.md"><img width="20px" src="https://iris-go.com/images/flag-france.svg" /></a> <a href="README_ZH_HANT.md"><img width="20px" src="https://iris-go.com/images/flag-taiwan.svg" /></a> <a href="README_ZH_HANS.md"><img width="20px" src="https://iris-go.com/images/flag-china.svg" /></a> <a href="README_ES.md"><img width="20px" src="https://iris-go.com/images/flag-spain.png" /></a> <a href="README_FA.md"><img width="20px" src="https://iris-go.com/images/flag-iran.svg" /></a> <a href="README_RU.md"><img width="20px" src="https://iris-go.com/images/flag-russia.svg" /></a> <a href="README_KO.md"><img width="20px" src="https://iris-go.com/images/flag-south-korea.svg?v=12" /></a> <a href="README_PT_BR.md"><img width="20px" height="20px" src="https://iris-go.com/images/flag-brazil.svg" /></a> <a href="README_VN.md"><img width="20px" height="20px" src="https://iris-go.com/images/flag-vietnam.svg" /></a>

[![build status](https://img.shields.io/github/actions/workflow/status/kataras/iris/ci.yml?branch=main&style=for-the-badge)](https://github.com/kataras/iris/actions/workflows/ci.yml) [![release](https://img.shields.io/badge/release%20-v14.0-0077b3.svg?style=for-the-badge)](https://github.com/kataras/iris/releases) [![view examples](https://img.shields.io/badge/examples%20-59-a83adf.svg?style=for-the-badge&logo=go)](_examples/README.md) [![godocs](https://img.shields.io/badge/go-%20docs-488AC7.svg?style=for-the-badge)](https://pkg.go.dev/github.com/kataras/iris/v14@v14.0.0) [![chat](https://img.shields.io/gitter/room/iris_go/community.svg?color=cc2b5e&logo=gitter&style=for-the-badge)](https://gitter.im/iris_go/community) [![donate](https://img.shields.io/badge/support-Iris-blue.svg?style=for-the-badge&logo=paypal)](https://iris-go.com/donate)

Το Iris είναι ένα γρήγορο web framework για τη Go, με όλα τα απαραίτητα ενσωματωμένα. Δίνει στο επόμενο API ή website σας εκφραστικό routing με typed παραμέτρους διαδρομής, ένα κεντρικό σημείο για τη διαχείριση σφαλμάτων, dependency injection που κρατά τον κώδικα εύκολο στο testing, και προεπιλογές παραγωγής που είναι ασφαλείς από την πρώτη στιγμή.

Η **έκδοση 14** είναι η μεγαλύτερη κυκλοφορία στην ιστορία του project. Το framework ξαναχτίστηκε γύρω από τα Go generics: ένας application builder που ελέγχεται κατά τη μεταγλώττιση, typed βοηθητικές συναρτήσεις αιτημάτων που κάνουν το validation για εσάς, controllers που επιλύονται από το dependency container, και μια ροή σφαλμάτων που ρυθμίζετε μία φορά. Το [HISTORY.md](HISTORY.md) καταγράφει κάθε νέο στοιχείο, ενώ ο [Οδηγός Μετάβασης](MIGRATION.md) φέρνει εφαρμογές v12 στη νέα έκδοση σε μία συνεδρία.

> 🤖 **Ρωτήστε την AI για το Iris.** Το δωρεάν [Code Wiki για το Iris](https://codewiki.google/github.com/kataras/iris) της Google παράγεται από αυτό το αποθετήριο και παραμένει συγχρονισμένο μαζί του, commit προς commit. Περιηγηθείτε στην αρχιτεκτονική με διαγράμματα, ή κάντε στο chat του ερωτήσεις σε απλή γλώσσα για τον κώδικα και λάβετε απαντήσεις θεμελιωμένες στον πραγματικό πηγαίο κώδικα.

Μάθετε τι [λένε οι άλλοι για το Iris](https://www.iris-go.com/#review) και δώστε ένα **[αστέρι](https://github.com/kataras/iris/stargazers)** σε αυτό το έργο ανοιχτού κώδικα για να στηρίξετε την ανάπτυξή του. Σας ενδιαφέρει η ταχύτητα; Δείτε τα ανεξάρτητα [server benchmarks](https://github.com/kataras/server-benchmarks).

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

<details><summary>Η στοίβα παραγωγής, σε μία έκφραση</summary>

Οι πραγματικές εφαρμογές χρησιμοποιούν το `iris.NewBuilder`: μια αλυσίδα που ελέγχεται κατά τη μεταγλώττιση και συνδέει CORS, συμπίεση, access logs, ένα health endpoint, τον χάρτη σφαλμάτων σας, τις υπηρεσίες σας και τα API groups σας. Ο compiler απορρίπτει τη λάθος σειρά βημάτων.

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

Οι handlers μένουν στις έξι γραμμές, γιατί η αποκωδικοποίηση, το validation και η απόδοση σφαλμάτων γίνονται σε ένα, ρυθμισμένο σημείο:

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

Ο [Οδηγός Πρώτων Βημάτων](TUTORIAL.md) χτίζει μια πλήρη, δοκιμασμένη υπηρεσία πάνω σε αυτή τη στοίβα, σε ένα έγγραφο.

</details>

Όπως είπε κάποτε ένας [Go developer](https://twitter.com/dkuye/status/1532087942696554497), **το Iris σε καλύπτει ολόπλευρα και παραμένει σταθερό στο πέρασμα των χρόνων**.

Τι σας προσφέρει το Iris v14:

* Routing με typed παραμέτρους διαδρομής (`{id:uuid}`, `{n:int}`, `{addr:email}` και άλλες), προσαρμοσμένα macros και περιορισμούς. Η μη έγκυρη είσοδος απορρίπτεται πριν τρέξει ο handler σας.
* `iris.NewBuilder`: CORS, συμπίεση, καταγραφή αιτημάτων, health endpoint, αντιστοίχιση σφαλμάτων, υπηρεσίες και API groups συναρμολογημένα σε μία αλυσίδα ελεγμένη από τον compiler.
* Dependency injection: οι constructors δηλώνουν τι χρειάζονται, το container το παρέχει. Hooks εκκίνησης `Initializer`, αυτόματα closers τερματισμού.
* Οι βοηθητικές συναρτήσεις `rest` και ένας κεντρικός χάρτης σφαλμάτων: λιτοί handlers, κανονικά, μηχανικά αναγνώσιμα σφάλματα, τα εσωτερικά δεν διαρρέουν ποτέ στους clients.
* Payloads αιτημάτων που επικυρώνονται μόνα τους: υλοποιήστε τη `Validate() error` και η `ctx.BindJSON` την εκτελεί μετά την αποκωδικοποίηση.
* Τριάντα ένα ενσωματωμένα middleware: accesslog, API keys, basicauth, bodylimit, cache, CORS, hCaptcha και reCAPTCHA, JWT, monitor, pprof, rate limiting, recovery, request IDs, rewrite, sessions, versioning και άλλα.
* Authentication σε κάθε κλίμακα: Basic Auth μίας γραμμής, API keys, ένα generic JWT SDK, ένας πλήρης identity server και ένας OAuth2 provider.
* MVC controllers χτισμένοι πάνω στα Go generics.
* WebSockets, εξυπηρέτηση gRPC και API versioning.
* View engines: HTML, Django, Handlebars, Pug και άλλες μέσω του iris-contrib/views.
* Localization με πληθυντικούς και ανίχνευση γλώσσας, μαζί με sitemaps.
* Sessions με ασφαλείς προεπιλογές cookies (`Secure`, `SameSite=Lax` από προεπιλογή).
* Πλούσιες αποκρίσεις: JSON, JSONP, XML, YAML, Markdown, Protocol Buffers, MessagePack, content negotiation, streaming και Server-Sent Events.
* AI backends: το πακέτο `ai` αποδίδει και αναμεταδίδει μέσω HTTP τη streaming έξοδο των μοντέλων (text deltas, reasoning traces, tool-call fragments, usage). Δεν εξαρτάται από κανέναν provider και δεν περιέχει κανένα vendor SDK.
* Ένας HTTP client, το πακέτο `client`, για να καλείτε άλλες υπηρεσίες με τους ίδιους κανόνες JSON που χρησιμοποιεί η πλευρά του server.
* Συμπίεση αποκρίσεων: gzip, deflate, brotli, snappy, s2, zstd.
* TLS με Let's Encrypt, και ενσωματωμένο ngrok tunneling (opt-in: `import _ "github.com/kataras/iris/v14/core/server/ngrok"`) για να βγάλετε μια dev εφαρμογή στο διαδίκτυο σε δευτερόλεπτα.
* In-memory HTTP testing: δοκιμές ολόκληρης της εφαρμογής χωρίς ports, χωρίς δίκτυο και χωρίς sleeps.
* Ένα πλήρες βιβλίο που μπορείτε να διαβάσετε online και να κατεβάσετε δωρεάν, ένα πρακτικό tutorial και ένα ζωντανό AI wiki (δείτε το Μαθαίνοντας το Iris παρακάτω).
* Και το σημαντικότερο: παίρνετε γρήγορες απαντήσεις και υποστήριξη, από την πρώτη μέρα μέχρι σήμερα.

## 📖 Μαθαίνοντας το Iris

> Νέοι στο Iris; Ξεκινήστε με τον [Οδηγό Πρώτων Βημάτων](TUTORIAL.md): ένα έγγραφο που σας πηγαίνει από το `go mod init` σε ένα δοκιμασμένο REST API με δομή παραγωγής. Αναβαθμίζετε από τη v12; Δείτε τον [Οδηγό Μετάβασης](MIGRATION.md). Ερωτήσεις σε οποιοδήποτε σημείο; Ρωτήστε το δωρεάν **ζωντανό AI wiki** στο [codewiki.google/github.com/kataras/iris](https://codewiki.google/github.com/kataras/iris), που παράγεται από την Google από τον πηγαίο κώδικα του Iris και παραμένει συγχρονισμένο μαζί του, με ενσωματωμένο chat που απαντά σε ερωτήσεις για τον κώδικα.

### Εγκατάσταση

Η μόνη απαίτηση είναι η [γλώσσα προγραμματισμού Go](https://go.dev/dl/), έκδοση 1.27 ή νεότερη.

#### Δημιουργία νέου project

```sh
$ mkdir myapp
$ cd myapp
$ go mod init myapp
$ go get github.com/kataras/iris/v14@latest # or @v14.0.0
```

<details><summary>Εγκατάσταση σε υπάρχον project</summary>

```sh
$ cd myapp
$ go get github.com/kataras/iris/v14@latest
```

**Εκτέλεση**

```sh
$ go mod tidy
$ go run .
```

</details>

![](https://www.iris-go.com/images/gifs/install-create-iris.gif)

Το Iris περιέχει εκτενή και αναλυτική **[τεκμηρίωση](https://www.iris-go.com/docs)** που κάνει εύκολο το ξεκίνημα με το framework.

Για πιο λεπτομερή τεχνική τεκμηρίωση μπορείτε να επισκεφθείτε τα [godocs](https://pkg.go.dev/github.com/kataras/iris/v14@v14.0.0) μας. Για μια πλήρη, πρακτική εισαγωγή στην προτεινόμενη στοίβα της v14 διαβάστε τον [Οδηγό Πρώτων Βημάτων](TUTORIAL.md), και αν αναβαθμίζετε από τη v12 ακολουθήστε τον [Οδηγό Μετάβασης](MIGRATION.md).

Προτιμάτε να μαθαίνετε από κώδικα; Τα [**59 παραδείγματα**](_examples/README.md) ξεκινούν από την πρώτη διαδρομή και φτάνουν στο dependency injection, στον έλεγχο ταυτότητας, στο testing και στα σημεία επέκτασης του framework. Καθένα από αυτά μεταγλωττίστηκε και δοκιμάστηκε πάνω από HTTP πριν γίνει commit.

### Το Βιβλίο του Iris

<a href="https://iris-go.com/book/"> <img alt="Book cover" src="https://iris-go.com/images/iris-book-cover-sm.jpg?v=14" /> </a>

Το πλήρες βιβλίο έχει έναν πρόλογο, 24 κεφάλαια και έναν επίλογο, με κάθε παράδειγμα κώδικα επαληθευμένο απέναντι στον πηγαίο κώδικα του framework. Κάθε κεφάλαιο ανοίγει με ένα υπόβαθρο σε απλή γλώσσα για τους νεοφερμένους και κλείνει με επιλεγμένη πρόσθετη βιβλιογραφία. Κάθε κεφάλαιο διαβάζεται ελεύθερα στο [iris-go.com/docs](https://iris-go.com/docs/): η [σελίδα του βιβλίου](https://iris-go.com/book/) έχει τα πλήρη περιεχόμενα.

Το PDF είναι κι αυτό δωρεάν. [Κατεβάστε το απευθείας](https://iris-go.com/iris-book.pdf): δεν υπάρχει φόρμα να συμπληρώσετε ούτε κάτι να ζητήσετε. Αν το βιβλίο σας βοηθήσει, με μια [χορηγία στο project](https://iris-go.com/donate/) παίρνετε μέρος στην ανάπτυξη του Iris.

[![follow author on twitter](https://img.shields.io/twitter/follow/makismaropoulos?color=3D8AA3&logoColor=3D8AA3&style=for-the-badge&logo=twitter)](https://twitter.com/intent/follow?screen_name=makismaropoulos)

[![follow Iris web framework on twitter](https://img.shields.io/twitter/follow/iris_framework?color=ee7506&logoColor=ee7506&style=for-the-badge&logo=twitter)](https://twitter.com/intent/follow?screen_name=iris_framework)

[![follow Iris web framework on facebook](https://img.shields.io/badge/Follow%20%40Iris.framework-569-2D88FF.svg?style=for-the-badge&logo=facebook)](https://www.facebook.com/iris.framework)

## 🤖 Εργαλεία AI για το Iris

Δύο εργαλεία AI γνωρίζουν συγκεκριμένα αυτό το framework, πέρα από όσα ξέρει ένα γενικό μοντέλο για τη Go.

Το [ζωντανό AI wiki](https://codewiki.google/github.com/kataras/iris) είναι δωρεάν και τρέχει στον browser: περιηγηθείτε στην αρχιτεκτονική ή ρωτήστε το chat του για τον κώδικα.

Το [Plexon AI](https://plexon.ai) είναι μια desktop εφαρμογή για Windows, macOS και Linux, φτιαγμένη από τον δημιουργό του Iris. Ενσωματώνει το skill του Iris ([iris-go.com/ai](https://iris-go.com/ai/)): το API της v14, την αλυσίδα του builder, τους `rest` helpers, τη χαρτογράφηση σφαλμάτων, το SDK πιστοποίησης και τα μοτίβα δοκιμών, σε είκοσι ένα έγγραφα αναφοράς που ο βοηθός φορτώνει όποτε τα χρειάζεται ενώ γράφει τον κώδικά σας. Ενεργοποιήστε το skill από το πάνελ Skills ή εγκαταστήστε την [persona Software Developer](https://plexon.ai/personas/software-developer/), που το ενεργοποιεί μαζί με τα υπόλοιπα εργαλεία της. [Κατεβάστε το Plexon AI](https://plexon.ai/download/).

Για τους υποστηρικτές, τη συνεισφορά, την πολιτική ασφαλείας και την άδεια χρήσης, δείτε το [αγγλικό README](README.md).

Αυτό το project αδειοδοτείται υπό την [άδεια BSD 3-clause](LICENSE), όπως ακριβώς και το ίδιο το project της Go.
