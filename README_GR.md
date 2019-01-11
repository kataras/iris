# Iris Web Framework <a href="README.md"> <img width="20px" src="https://iris-go.com/images/flag-unitedkingdom.svg?v=10" /></a> <a href="README_ZH.md"> <img width="20px" src="https://iris-go.com/images/flag-china.svg?v=10" /></a> <a href="README_RU.md"><img width="20px" src="https://iris-go.com/images/flag-russia.svg?v=10" /></a> <a href="README_ID.md"> <img width="20px" src="https://iris-go.com/images/flag-indonesia.svg?v=10" /></a> <a href="README_PT_BR.md"><img width="20px" src="https://iris-go.com/images/flag-pt-br.svg?v=10" /></a> <a href="README_JPN.md"><img width="20px" src="https://iris-go.com/images/flag-japan.svg?v=10" /></a>

<a href="https://iris-go.com"> <img align="right" width="169px" src="https://iris-go.com/images/icon.svg?v=a" title="logo created by @merry.dii" /> </a>

[![build status](https://img.shields.io/travis/kataras/iris/master.svg?style=flat-square)](https://travis-ci.org/kataras/iris)<!-- [![release](https://img.shields.io/github/release/kataras/iris.svg?style=flat-square)](https://github.com/kataras/iris/releases)--> [![report card](https://img.shields.io/badge/report%20card-a%2B-ff3333.svg?style=flat-square)](http://goreportcard.com/report/kataras/iris) [![vscode-iris](https://img.shields.io/badge/ext%20-vscode-0c77e3.svg?style=flat-square)](https://marketplace.visualstudio.com/items?itemName=kataras2006.iris)<!--[![github closed issues](https://img.shields.io/github/issues-closed-raw/kataras/iris.svg?style=flat-square)](https://github.com/kataras/iris/issues?q=is%3Aissue+is%3Aclosed)--> [![chat](https://img.shields.io/badge/community-%20chat-00BCD4.svg?style=flat-square)](https://kataras.rocket.chat/channel/iris) [![view examples](https://img.shields.io/badge/learn%20by-examples-0077b3.svg?style=flat-square)](https://github.com/kataras/iris/tree/master/_examples/routing) [![release](https://img.shields.io/badge/release%20-v11.1-0077b3.svg?style=flat-square)](https://github.com/kataras/iris/releases)

Το Iris είναι ένα γρήγορο, απλό αλλά και πλήρως λειτουργικό και πολύ αποδοτικό web framework για τη Go.

Το Iris παρέχει ένα όμορφα εκφραστικό και εύχρηστο υπόβαθρο για την επόμενη σας ιστοσελίδα ή API.

Επιτέλους, ένα πραγματικά ισάξιο (και με το παραπάνω) expressjs web framework για τη γλώσσα προγραμματισμού Go.

Μάθετε τι [λένε οι άλλοι για το Iris](#%CE%A5%CF%80%CE%BF%CF%83%CF%84%CE%AE%CF%81%CE%B9%CE%BE%CE%B7) και [δώστε ένα αστέρι](https://github.com/kataras/iris/stargazers) στο github repository για να μένετε [πάντα ενημερωμένοι](https://facebook.com/iris.framework).

## Yποστηρικτές

Eυχαριστούμε όλους τους υποστηρικτές μας! 🙏 [Γίνετε ένας από αυτούς](https://iris-go.com/donate)

<a href="https://iris-go.com/donate" target="_blank"><img src="https://iris-go.com/backers.svg?v=2"/></a>

```sh
$ cat example.go
```

```go
package main

import "github.com/kataras/iris"

func main() {
    app := iris.New()
    // Εδώ φορτώνουμε όλα τα templates από τον
    // φάκελο "./views"
    // όπου το extension είναι ".html" και αναλύουμε
    // τα αρχεία αυτά βάση του `html/template` πακέτου.
    app.RegisterView(iris.HTML("./views", ".html"))

    // Method:    GET
    // Resource:  http://localhost:8080
    app.Get("/", func(ctx iris.Context) {
        // Όπου {{.message}} εμφάνισε "Hello world!"
        ctx.ViewData("message", "Hello world!")
        // Εμφάνισε το σχετικό αρχείο "./views/hello.html"
        ctx.View("hello.html")
    })

    // Method:    GET
    // Resource:  http://localhost:8080/user/42
    //
    // Θέλετε να χρησημοποιήσετε regex expressions;
    // Εύκολο,
    // απλά δηλώστε τον τύπο της παραμέτρου ως 'string'
    // ο οποίος δέχετε κάθε τιμή και κάντε χρήση
    // της `regexp` macro function, για παράδειγμα:
    // app.Get("/user/{id:string regexp(^[0-9]+$)}")
    app.Get("/user/{id:long}", func(ctx iris.Context) {
        userID, _ := ctx.Params().GetInt64("id")
        ctx.Writef("User ID: %d", userID)
    })

    // Εδώ αρχίζουμε τον server χρησιμοποιώντας την
    // τοπική διεύθυνση δικτύου με πόρτα την 8080.
    app.Run(iris.Addr(":8080"))
}
```

> Μάθετε περισσότερα για τους τύπους παραμέτρων διαδρομής(routing) πατώντας [εδώ](_examples/routing/dynamic-path/main.go#L31)

```html
<!-- αρχείο: ./views/hello.html -->
<html>
<head>
    <title>Hello Page</title>
</head>
<body>
    <h1>{{.message}}</h1>
</body>
</html>
```

```sh
$ go run example.go
Now listening on: http://localhost:8080
Application Started. Press CTRL+C to shut down.
_
```

## Εγκατάσταση

Η μόνη απαίτηση είναι η [Go Γλώσσα Προγραμματισμού](https://golang.org/dl/)

```sh
$ go get -u github.com/kataras/iris
```

Το Iris εκμεταλλεύεται τη λεγόμενη λειτουργία [vendor directory](https://docs.google.com/document/d/1Bz5-UB7g2uPBdOx-rw5t9MxJwkfpx90cqG9AFL0JAYo). Παίρνετε πλήρως αναπαραγωγίσιμα builds, καθώς αυτή η μέθοδος προστατεύει από τις upstream μετονομασίες και διαγραφές.

[![Iris vs .NET Core(C#) vs Node.js (Express)](https://iris-go.com/images/benchmark-new-gray.png)](_benchmarks/README_UNIX.md)

_Η τελευταία ενημέρωση έγινε την [Τρίτη, 21 Νοεμβρίου του 2017](_benchmarks/README_UNIX.md)_

<details>
<summary>Στοιχεία αναφοράς από τρίτες πηγές σε σχέση με τα υπόλοιπα web frameworks</summary>

![Comparison with other frameworks](https://raw.githubusercontent.com/smallnest/go-web-framework-benchmark/4db507a22c964c9bc9774c5b31afdc199a0fe8b7/benchmark.png)

</details>

## Υποστήριξη

- To [HISTORY](HISTORY_GR.md#fr-11-january-2019--v1111) αρχείο είναι ο καλύτερος σας φίλος, περιέχει πληροφορίες σχετικά με τις τελευταίες λειτουργίες(features) και αλλαγές
- Μήπως τυχαίνει να βρήκατε κάποιο bug; Δημοσιεύστε το στα [github issues](https://github.com/kataras/iris/issues)
- Έχετε οποιεσδήποτε ερωτήσεις ή πρέπει να μιλήσετε με κάποιον έμπειρο για την επίλυση ενός προβλήματος σε πραγματικό χρόνο; Ελάτε μαζί μας στην [συνομιλία κοινότητας](https://chat.iris-go.com)
- Συμπληρώστε την αναφορά εμπειρίας χρήστη κάνοντας κλικ [εδώ](https://docs.google.com/forms/d/e/1FAIpQLSdCxZXPANg_xHWil4kVAdhmh7EBBHQZ_4_xSZVDL-oCC_z5pA/viewform?usp=sf_link)
- Σας αρέσει το Iris; Τιτιβίστε κάτι για αυτό! Άνθρωποι από ολόκληρο τον πλανήτη έχουνε μιλήσει για αυτό ακριβώς:

<a href="https://twitter.com/gelnior/status/769100480706379776"> 
    <img src="https://comments.iris-go.com/comment27_mini.png" width="350px">
</a>

<a href="https://twitter.com/MeAlex07/status/822799954188075008"> 
    <img src="https://comments.iris-go.com/comment28_mini.png" width="350px">
</a>

<a href="https://twitter.com/_mgale/status/818591490305761280"> 
    <img src="https://comments.iris-go.com/comment29_mini.png" width="350px">
</a>
<a href="https://twitter.com/VeayoX/status/813273328550973440"> 
    <img src="https://comments.iris-go.com/comment30_mini.png" width="350px">
</a>

<a href="https://twitter.com/pvsukale/status/745328224876408832"> 
    <img src="https://comments.iris-go.com/comment31_mini.png" width="350px">
</a>

<a href="https://twitter.com/blainsmith/status/745338092211560453"> 
    <img src="https://comments.iris-go.com/comment32_mini.png" width="350px">
</a>

<a href="https://twitter.com/tjbyte/status/758287014210867200"> 
    <img src="https://comments.iris-go.com/comment33_mini.png" width="350px">
</a>

<a href="https://twitter.com/tangzero/status/751050577220698112"> 
    <img src="https://comments.iris-go.com/comment34_mini.png" width="350px">
</a>

<a href="https://twitter.com/tjbyte/status/758287244947972096"> 
    <img src="https://comments.iris-go.com/comment33_2_mini.png" width="350px">
</a>

<a href="https://twitter.com/ferarias/status/902468752364773376"> 
    <img src="https://comments.iris-go.com/comment41.png" width="350px">
</a>

<br/><br/>

Για περισσότερες πληροφορίες σχετικά με τη συμβολή στο Iris, διαβάστε το [CONTRIBUTING.md](CONTRIBUTING.md) αρχείο.

[Κατάλογος όλων των Συνεργατών](https://github.com/kataras/iris/graphs/contributors)

## Μάθηση

Πρώτα απ 'όλα, ο πιο σωστός τρόπος για να ξεκινήσετε με ένα web framework είναι να μάθετε τα βασικά της γλώσσας προγραμματισμού και των τυπικών της δυνατοτήτων `http`, αν η εφαρμογή σας είναι ένα πολύ απλό προσωπικό έργο χωρίς απαιτήσεις επιδόσεων και συντηρησιμότητας, ίσως να θέλετε να προχωρήσετε μόνο με τα τυπικά πακέτα, εαν οχι τότε ακολουθήστε τις παρακάτω οδηγίες:

- Πλοηγηθείτε μέσω των **100+1** **[παραδειγμάτων](_examples)** και μερικές [απλές εφαρμογές για αρχάριους](#iris-starter-kits) που δημιουργήσαμε για εσάς
- Διαβάστε τα [godocs](https://godoc.org/github.com/kataras/iris) για οποιαδήποτε λεπτομέρεια
- Ετοιμάστε ένα φλιτζάνι καφέ ή τσάι, ό,τι σας ευχαριστεί περισσότερο και διαβάστε κάποια [άρθρα](#articles) που βρήκαμε για εσάς

### Iris starter kits

<!-- table form 
| Description | Link |
| -----------|-------------|
| Hasura hub starter project with a ready to deploy golang helloworld webapp with IRIS! | https://hasura.io/hub/project/hasura/hello-golang-iris |
| A basic web app built in Iris for Go |https://github.com/gauravtiwari/go_iris_app |
| A mini social-network created with the awesome Iris💖💖 | https://github.com/iris-contrib/Iris-Mini-Social-Network |
| Iris isomorphic react/hot reloadable/redux/css-modules starter kit | https://github.com/iris-contrib/iris-starter-kit |
| Demo project with react using typescript and Iris | https://github.com/ionutvilie/react-ts |
| Self-hosted Localization Management Platform built with Iris and Angular | https://github.com/iris-contrib/parrot |
| Iris + Docker and Kubernetes | https://github.com/iris-contrib/cloud-native-go |
| Quickstart for Iris with Nanobox | https://guides.nanobox.io/golang/iris/from-scratch |
-->

1. [snowlyg/IrisApiProject: Iris + gorm + jwt + sqlite3](https://github.com/snowlyg/IrisApiProject) **NEW-Chinese**
2. [yz124/superstar: Iris + xorm to implement the star library](https://github.com/yz124/superstar) **NEW-Chinese**
3. [jebzmos4/Iris-golang: A basic CRUD API in golang with Iris](https://github.com/jebzmos4/Iris-golang)
4. [gauravtiwari/go_iris_app: A basic web app built in Iris for Go](https://github.com/gauravtiwari/go_iris_app)
5. [A mini social-network created with the awesome Iris💖💖](https://github.com/iris-contrib/Iris-Mini-Social-Network)
6. [Iris isomorphic react/hot reloadable/redux/css-modules starter kit](https://github.com/iris-contrib/iris-starter-kit)
7. [ionutvilie/react-ts: Demo project with react using typescript and Iris](https://github.com/ionutvilie/react-ts)
8. [Self-hosted Localization Management Platform built with Iris and Angular](https://github.com/iris-contrib/parrot)
9. [Iris + Docker and Kubernetes](https://github.com/iris-contrib/cloud-native-go)
10. [nanobox.io: Quickstart for Iris with Nanobox](https://guides.nanobox.io/golang/iris/from-scratch)
11. [hasura.io: A Hasura starter project with a ready to deploy Golang hello-world web app with IRIS](https://hasura.io/hub/project/hasura/hello-golang-iris)

> Έχετε χτίσει κάτι παρόμοιο; [Ενημέρωσέ μας](https://github.com/kataras/iris/pulls)!

### Middleware

Το Iris έχει μια μεγάλη συλλογή Handlers[[1]](middleware/)[[2]](https://github.com/iris-contrib/middleware) που μπορείτε να χρησιμοποιήσετε μέσα στις εφαρμογές σας. Ωστόσο, δεν περιορίζεστε σε αυτά - είστε ελεύθεροι να χρησιμοποιήσετε οποιοδήποτε μεσαίο λογισμικό τρίτου μέρους που είναι συμβατό με το [net/http](https://golang.org/pkg/net/http/) πακέτο, [_examples/convert-handlers](_examples/convert-handlers) θα σας δείξουν τον δρόμο.

Το Iris, σε αντίθεση με τα άλλα, είναι 100% συμβατό με τα πρότυπα και γι 'αυτό η πλειοψηφία των μεγάλων εταιρειών που προσαρμόζονται στην Go, όπως ένα πολύ γνωστό τηλεοπτικό δίκτυο των ΗΠΑ, εμπιστεύονται το Iris, και αυτό γιατί είναι πάντα ενημερωμένο και ευθυγραμμισμένο με το πακέτο `net/http` το οποίο εκσυγχρονίζεται από τους συγγραφέες(authors) της Go σε κάθε νέα έκδοση της, για πάντα.

### Articles

* [CRUD REST API in Iris (a framework for golang)](https://medium.com/@jebzmos4/crud-rest-api-in-iris-a-framework-for-golang-a5d33652401e)
* [A Todo MVC Application using Iris and Vue.js](https://hackernoon.com/a-todo-mvc-application-using-iris-and-vue-js-5019ff870064)
* [A Hasura starter project with a ready to deploy Golang hello-world web app with IRIS](bit.ly/2lmKaAZ)
* [Top 6 web frameworks for Go as of 2017](https://blog.usejournal.com/top-6-web-frameworks-for-go-as-of-2017-23270e059c4b)
* [Iris Go Framework + MongoDB](https://medium.com/go-language/iris-go-framework-mongodb-552e349eab9c)
* [How to build a file upload form using DropzoneJS and Go](https://hackernoon.com/how-to-build-a-file-upload-form-using-dropzonejs-and-go-8fb9f258a991)
* [How to display existing files on server using DropzoneJS and Go](https://hackernoon.com/how-to-display-existing-files-on-server-using-dropzonejs-and-go-53e24b57ba19)
* [Iris, a modular web framework](https://medium.com/@corebreaker/iris-web-cd684b4685c7)
* [Go vs .NET Core in terms of HTTP performance](https://medium.com/@kataras/go-vs-net-core-in-terms-of-http-performance-7535a61b67b8)
* [Iris Go vs .NET Core Kestrel in terms of HTTP performance](https://hackernoon.com/iris-go-vs-net-core-kestrel-in-terms-of-http-performance-806195dc93d5)
* [How to Turn an Android Device into a Web Server](https://twitter.com/ThePracticalDev/status/892022594031017988)
* [Deploying a Iris Golang app in hasura](https://medium.com/@HasuraHQ/deploy-an-iris-golang-app-with-backend-apis-in-minutes-25a559bf530b)
* [A URL Shortener Service using Go, Iris and Bolt](https://medium.com/@kataras/a-url-shortener-service-using-go-iris-and-bolt-4182f0b00ae7)

### Εκμάθηση μέσω video

* [Daily Coding - Web Framework Golang: Iris Framework]( https://www.youtube.com/watch?v=BmOLFQ29J3s) από WarnabiruTV, πηγή: youtube, κόστος: **ΔΩΡΕΑΝ**
* [Tutorial Golang MVC dengan Iris Framework & Mongo DB](https://www.youtube.com/watch?v=uXiNYhJqh2I&list=PLMrwI6jIZn-1tzskocnh1pptKhVmWdcbS) (19 videos ως τωρα) από Musobar Media, πηγή: youtube, κόστος: **ΔΩΡΕΑΝ**
* [Go/Golang 27 - Iris framework : Routage de base](https://www.youtube.com/watch?v=rQxRoN6ub78) από stephgdesign, πηγή: youtube, κόστος: **ΔΩΡΕΑΝ**
* [Go/Golang 28 - Iris framework : Templating](https://www.youtube.com/watch?v=nOKYV073S2Y) από stephgdesignn, πηγή: youtube, κόστος: **ΔΩΡΕΑΝ**
* [Go/Golang 29 - Iris framework : Paramètres](https://www.youtube.com/watch?v=K2FsprfXs1E) από stephgdesign, πηγή: youtube, κόστος: **ΔΩΡΕΑΝ**
* [Go/Golang 30 - Iris framework : Les middelwares](https://www.youtube.com/watch?v=BLPy1So6bhE) από stephgdesign, πηγή: youtube, κόστος: **ΔΩΡΕΑΝ**
* [Go/Golang 31 - Iris framework : Les sessions](https://www.youtube.com/watch?v=RnBwUrwgEZ8) από stephgdesign, πηγή: youtube, κόστος: **ΔΩΡΕΑΝ**

### Προσληφθείτε

Υπάρχουν πολλές νεοσύστατες εταιρείες που αναζητούν Go web developers με εμπειρία Iris ως απαίτηση, ψάχνουμε καθημερινά και δημοσιεύουμε αυτές τις πληροφορίες μέσω της [σελίδας μας στο facebook](https://www.facebook.com/iris.framework), κάντε like για να λαμβάνετε ειδοποιήσεις, έχουμε ήδη δημοσιεύσει ορισμένες από αυτές(τις θέσεις εργασίας).

## License

Το Iris διαθέτει άδεια βάσει του [3-Clause BSD License](LICENSE). Το Iris είναι 100% δωρεάν και  ανοιχτού κώδικα λογισμικό.

Για τυχόν ερωτήσεις σχετικά με την άδεια παρακαλώ στείλτε [e-mail](mailto:kataras2006@hotmail.com?subject=Iris%20License).
