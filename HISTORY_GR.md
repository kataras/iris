# Ιστορικό <a href="HISTORY.md"> <img width="20px" src="https://iris-go.com/images/flag-unitedkingdom.svg?v=10" /></a> <a href="HISTORY_ZH.md"> <img width="20px" src="https://iris-go.com/images/flag-china.svg?v=10" /></a> <a href="HISTORY_ID.md"> <img width="20px" src="https://iris-go.com/images/flag-indonesia.svg?v=10" /></a>

### Ψάχνετε για δωρεάν υποστήριξη σε πραγματικό χρόνο;

    https://github.com/kataras/iris/issues
    https://chat.iris-go.com

### Ψάχνετε για προηγούμενες εκδόσεις;

    https://github.com/kataras/iris/releases

### Πρέπει να αναβαθμίσω το Iris μου;

Οι προγραμματιστές δεν αναγκάζονται να αναβαθμίσουν αν δεν το χρειάζονται πραγματικά. Αναβαθμίστε όποτε αισθάνεστε έτοιμοι.

> Το Iris εκμεταλλεύεται τη λεγόμενη λειτουργία [vendor directory](https://docs.google.com/document/d/1Bz5-UB7g2uPBdOx-rw5t9MxJwkfpx90cqG9AFL0JAYo). Παίρνετε πλήρως αναπαραγωγίσιμα builds, καθώς αυτή η μέθοδος προστατεύει από τις upstream μετονομασίες και διαγραφές.

**Πώς να αναβαθμίσετε**: Ανοίξτε την γραμμή εντολών σας και εκτελέστε αυτήν την εντολή: `go get -u github.com/kataras/iris`  ή αφήστε το αυτόματο updater να το κάνει αυτό για σας.

# Fr, 11 January 2019 | v11.1.1

Πατήστε [εδώ](https://github.com/kataras/iris/blob/master/HISTORY.md#fr-11-january-2019--v1111) για να διαβάσετε στα αγγλικά.

# Su, 18 November 2018 | v11.1.0

Πατήστε [εδώ](https://github.com/kataras/iris/blob/master/HISTORY.md#su-18-november-2018--v1110) για να διαβάσετε στα αγγλικά για το νέο "versioning" feature.

# Fr, 09 November 2018 | v11.0.4

Πατήστε [εδώ](https://github.com/kataras/iris/blob/master/HISTORY.md#fr-09-november-2018--v1104) για να διαβάσετε στα αγγλικά τις αλλαγές που φέρνει το τελευταίο patch για την έκδοση 11.

# Tu, 30 October 2018 | v11.0.2

Πατήστε [εδώ](https://github.com/kataras/iris/blob/master/HISTORY.md#tu-30-october-2018--v1102) για να διαβάσετε στα αγγλικά μια σημαντική διόρθωση για τα x32 machines που φέρνει το τελευταίο patch για την έκδοση 11.

# Su, 28 October 2018 | v11.0.1

Πατήστε [εδώ](https://github.com/kataras/iris/blob/master/HISTORY.md#su-28-october-2018--v1101) για να διαβάσετε στα αγγλικά τις αλλαγές και το νέο feature που φέρνει το τελευταίο patch για την έκδοση 11.

# Su, 21 October 2018 | v11.0.0

Πατήστε [εδώ](https://github.com/kataras/iris/blob/master/HISTORY.md#su-21-october-2018--v1100) για να διαβάσετε στα αγγλικά τις αλλαγές και τα νέα features που φέρνει νεότερη έκδοση του Iris, version 11.

# Sat, 11 August 2018 | v10.7.0

Είμαι στην, πραγματικά, ευχάριστη θέση να σας ανακοινώσω το πρώτο στάδιο της σταθερής κυκλοφορίας της έκδοσης **10.7** του Iris Web Framework.

Η έκδοση 10.7.0 είναι μέρος των [επίσημων διανομών μας](https://github.com/kataras/iris/releases).

Αυτή η έκδοση δεν περιέχει αλλαγές που μπορούν να αλλάξουν ριζικά τα υπάρχοντα προγράμματα που χτίστηκαν με βάση παλιότερες εκδόσεις του Iris. Οι προγραμματιστές μπορούν να αναβαθμίσουν με απόλυτη ασφάλεια.

Διαβάστε παρακάτω τις αλλαγές και τις βελτιώσεις στον εσωτερικό άβυσσο του Iris. Επιπλέον έχουμε ακόμα περισσότερα παραδείγματα για αρχάριους στην κοινότητα μας.

## Νέα παραδείγματα

- [Iris + WebAssemply = 💓](_examples/webassembly/basic/main.go) **προυποθέτει την έκδοση go11.beta και αφεξ**
- [Server-Sent Events](_examples/http_responsewriter/sse/main.go)
- [Struct Validation on context.ReadJSON](_examples/http_request/read-json-struct-validation/main.go)
- [Extract referrer from "referer" header or URL query parameter](_examples/http_request/extract-referer/main.go)
- [Hero Sessions](_examples/hero/sessions)
- [Yet another dependency injection example with hero](_examples/hero/smart-contract/main.go)
- [Writing an API for the Apache Kafka](_examples/tutorial/api-for-apache-kafka)

> Επίσης, όλα τα "sessions" παραδείγματα έχουν προσαρμοστεί ώστε να περιέχουν την επιλογή `AllowReclaim: true`

## kataras/iris/websocket

- Αλλαγή του "connection list" από απλή λίστα σε `sync.Map`, έγινε με: [αυτό](https://github.com/kataras/iris/commit/5f16704f45bedd767527eadf411cf9bc0f8edaee) και [αυτό το commit](https://github.com/kataras/iris/commit/16b30e8eed1406c61abc01282120870bd9fa31d8)
- Προσθήκη του `iris-ws.js` αρχείου στο διάσημο CDN https://cdnjs.com με [αυτό το PR](https://github.com/kataras/iris/pull/1053) από τον [Dibyendu Das](https://github.com/dibyendu)

## kataras/iris/core/router

- Προσθήκη κάποιων `json` field tags και νέα functions όπως τα `ChangeMethod`, `SetStatusOffline` και `RestoreStatus` στη `Route` δομή, προσοχή, για να υσχίσουν αυτών των ειδών οι αλλαγές προυποθέτουν από εσας το κάλεσμα του `Router/Application.RefreshRouter()`. (δεν συνιστάται απόλυτα αλλά είναι χρήσιμο για ειδικά κατασκευασμένες ιστοσελίδες τρίτων που μπορεί να υπάρχουν εκει έξω για διαχείριση ενός Web Server που χτίστηκε με Iris)
- Προσθήκη του `GetRoutesReadOnly` function στην `APIBuilder` δομή

## kataras/iris/context

- Προσθήκη των `GetReferrer`, `GetContentTypeRequested` και `URLParamInt32Default` functions
- Εισαγωγή των `Trace`, `Tmpl` και `MainHandlerName` functions στη διεπαφή(interface) `RouteReadOnly`
- Προσθήκη του `OnConnectionClose` function, χρησημοποείται για να καλεί κάποιο function όταν η "underline tcp connection" διακοπεί, εξαιρετικά χρήσιμο για SSE ή και άλλες παρόμοιες υλοποιήσεις μέσα σε έναν Iris Handler -- και του `OnClose` function το οποίο είναι σαν να καλείτε τα `OnConnectionClose(myFunc)` και `defer myFunc()` στον ίδιο Iris Handler [*](https://github.com/kataras/iris/commit/6898c2f755a0e22aa42e3b1799e29c857777a6f9)

Αυτή η έκδοση περιέχει επίσης κάποιες δευτερεύουσες γραμματικής και τυπογραφικού λάθους διορθώσεις και πιό ευανάγνωστα σχόλια για το [godoc](https://godoc.org/github.com/kataras/iris).

## Βιομηχανία

Βρίσκομαι στην ευχάριστη αυτή θέση να σας ανακοινώσω ότι το Iris έχει επιλεγεί ως το κύριο αναπτυξιακό kit(main development kit) για οκτώ μεσαίου ως μεγάλου μεγέθους εταιρείες  και μια νέα πολύ ελπιδοφόρα startup με βάση την Ινδία.

Θέλω να σας ευχαριστήσω όλους, **καθέ μία και έναν ξεχωριστά**, για την αδιάκοπη αυτή υποστήριξη και την εμπιστοσύνη που μου δείξατε όλα αυτά τα χρόνια, ειδικά φέτος, παρά τις φήμες και τις διάφορες δυσφήσεις που είχαμε υποστεί αδίκως από τον ανελέητο ανταγωνισμό.

# Tu, 05 June 2018 | v10.6.6

This history entry is not translated yet to the Greek language yet, please refer to the english version of the [HISTORY entry](https://github.com/kataras/iris/blob/master/HISTORY.md#tu-05-june-2018--v1066) instead.

# Mo, 21 May 2018 | v10.6.5

This history entry is not translated yet to the Greek language yet, please refer to the english version of the [HISTORY entry](https://github.com/kataras/iris/blob/master/HISTORY.md#mo-21-may-2018--v1065) instead.

# We, 09 May 2018 | v10.6.4

- [διόρθωση του bug #995](https://github.com/kataras/iris/commit/62457279f41a1f157869a19ef35fb5198694fddb)
- [διόρθωση του bug #996](https://github.com/kataras/iris/commit/a11bb5619ab6b007dce15da9984a78d88cd38956)

# We, 02 May 2018 | v10.6.3

This history entry is not translated yet to the Greek language yet, please refer to the english version of the [HISTORY entry](https://github.com/kataras/iris/blob/master/HISTORY.md#we-02-may-2018--v1063) instead.

# Tu, 01 May 2018 | v10.6.2

This history entry is not translated yet to the Greek language yet, please refer to the english version of the [HISTORY entry](https://github.com/kataras/iris/blob/master/HISTORY.md#tu-01-may-2018--v1062) instead.

# We, 25 April 2018 | v10.6.1

This history entry is not translated yet to the Greek language yet, please refer to the english version of the [HISTORY entry](https://github.com/kataras/iris/blob/master/HISTORY.md#we-25-april-2018--v1061) instead.

# Su, 22 April 2018 | v10.6.0

This history entry is not translated yet to the Greek language yet, please refer to the english version of the [HISTORY entry](https://github.com/kataras/iris/blob/master/HISTORY.md#su-22-april-2018--v1060) instead.

# Sa, 24 March 2018 | v10.5.0

This history entry is not translated yet to the Greek language yet, please refer to the english version of the [HISTORY entry](https://github.com/kataras/iris/blob/master/HISTORY.md#sa-24-march-2018--v1050) instead.

# We, 14 March 2018 | v10.4.0

This history entry is not translated yet to the Greek language yet, please refer to the english version of the [HISTORY entry](https://github.com/kataras/iris/blob/master/HISTORY.md#we-14-march-2018--v1040) instead.

# Sa, 10 March 2018 | v10.3.0

This history entry is not translated yet to the Greek language yet, please refer to the english version of the [HISTORY entry](https://github.com/kataras/iris/blob/master/HISTORY.md#sa-10-march-2018--v1030) instead.

# Th, 15 February 2018 | v10.2.1

Διόρθωση το οποίο αφορά 404 not found errors στα αρχεία που σερβίρονται από τα `StaticEmbedded` και `StaticWeb` των υποτομεών(subdomains), όπως αναφέρθηκε πριν λίγο από τον [@speedwheel](https://github.com/speedwheel) μέσω [της σελίδας μας στο facebook](https://facebook.com/iris.framework).

This history entry is not yet translated to Greek. Please read [the english version instead](https://github.com/kataras/iris/blob/master/HISTORY.md#th-15-february-2018--v1021).

# Th, 08 February 2018 | v10.2.0

This history entry is not yet translated to Greek. Please read [the english version instead](https://github.com/kataras/iris/blob/master/HISTORY.md#th-08-february-2018--v1020).

# Tu, 06 February 2018 | v10.1.0

This history entry is not yet translated to Greek. Please read [the english version instead](https://github.com/kataras/iris/blob/master/HISTORY.md#tu-06-february-2018--v1010).

# Tu, 16 January 2018 | v10.0.2

## Ασφάλεια | `iris.AutoTLS`

**Όλοι οι servers πρέπει να αναβαθμιστούν σε αυτήν την έκδοση**, περιέχει διορθώσεις για το _tls-sni challenge_ το οποίο απενεργοποιήθηκε μερικές μέρες πριν από το letsencrypt.org το οποίο προκάλεσε σχεδόν όλα τα golang https-ενεργποιημένα servers να να μην είναι σε θέση να λειτουργήσουν, έτσι υποστήριξη για το _http-01 challenge_ προστέθηκε σαν αναπλήρωση. Πλέον ο διακομιστής δοκιμάζει όλες τις διαθέσιμες προκλήσεις(challenges) letsencrypt.

Διαβάστε περισσότερα:

- https://letsencrypt.status.io/pages/incident/55957a99e800baa4470002da/5a55777ed9a9c1024c00b241
- https://github.com/golang/crypto/commit/13931e22f9e72ea58bb73048bc752b48c6d4d4ac

# Mo, 15 January 2018 | v10.0.1

- διόρθωση του cache handler που δεν δούλευε όπως έπρεπε όταν γινόταν εγγραφή σε πάνω από ένα handler, παλιότερα ήταν ένα cache handler προς ένα route handler, τώρα το ίδιο handler μπορεί να καταχωρηθεί σε όσα route handlers θέλετε https://github.com/kataras/iris/pull/852, όπως είχε αναφερθεί στο https://github.com/kataras/iris/issues/850
- συγχώνευση PR https://github.com/kataras/iris/pull/862
- απαγόρευση της ταυτόχρονης προσπέλασης του `ExecuteWriter -> Load` όταν το `view#Engine##Reload` είναι true, όπως είχε ζητηθεί στο https://github.com/kataras/iris/issues/872
- αναβάθμιση του ενσωματωμένου πακέτου `golang/crypto` για να εφαρμοστεί η επιδιόρθωση για το [tls-sni challenge disabled](https://letsencrypt.status.io/pages/incident/55957a99e800baa4470002da/5a55777ed9a9c1024c00b241) https://github.com/golang/crypto/commit/13931e22f9e72ea58bb73048bc752b48c6d4d4ac (**σχετικό με το iris.AutoTLS**)

## Νέοι Υποστηρικτές

1. https://opencollective.com/cetin-basoz

## Νέες Μεταφράσεις

1. Aναβαθμίσεις για την Κινέζικη μετάφραση README_ZH.md (νέο) και HISTORY_ZH.md  από @Zeno-Code μέσω του https://github.com/kataras/iris/pull/858
2. Το Ρώσικο README_RU.md μεταφράστηκε από την @merrydii μέσω του https://github.com/kataras/iris/pull/857
3. Τα Ελληνικά README_GR.md και HISTORY_GR.md μεταφράστηκαν μέσω των https://github.com/kataras/iris/commit/8c4e17c2a5433c36c148a51a945c4dc35fbe502a#diff-74b06c740d860f847e7b577ad58ddde0 και https://github.com/kataras/iris/commit/bb5a81c540b34eaf5c6c8e993f644a0e66a78fb8

## Νέα Παραδείγματα

1. [MVC - Register Middleware](_examples/mvc/middleware)

## Νέα Άρθρα

1. [A Todo MVC Application using Iris and Vue.js](https://hackernoon.com/a-todo-mvc-application-using-iris-and-vue-js-5019ff870064)
2. [A Hasura starter project with a ready to deploy Golang hello-world web app with IRIS](bit.ly/2lmKaAZ)

# Mo, 01 January 2018 | v10.0.0

Πρέπει να ευχαριστήσουμε την [Κυρία Diana](https://www.instagram.com/merry.dii/) για το νέο μας [λογότυπο](https://iris-go.com/images/icon.svg)!

Μπορείτε να [επικοινωνήσετε](mailto:Kovalenkodiana8@gmail.com) μαζί της για οποιεσδήποτε σχετικές με το σχεδιασμό εργασίες ή να της στείλειτε ένα άμεσο μήνυμα μέσω [instagram](https://www.instagram.com/merry.dii/).

<p align="center">
<img width="145px" src="https://iris-go.com/images/icon.svg?v=a" />
</p>

Σε αυτή την έκδοση έχουμε πολλές εσωτερικές βελτιώσεις αλλά μόνο δύο μεγάλες αλλαγές και ένα μεγάλο χαρακτηριστικό, που ονομάζεται **hero**.

> Η νέα έκδοση προσθέτει 75 καινούρια commits, το PR βρίσκεται [εδώ](https://github.com/kataras/iris/pull/849). Παρακαλώ διαβάστε τις εσωτερικές αλλαγές αν σχεδιάζετε ένα web framework που βασίζεται στο Iris. Γιατί η έκδοση 9 παραλείφθηκε; Έτσι.

## Hero

Το νέο πακέτο [hero](hero) περιέχει χαρακτηριστικά για σύνδεση(binding) οποιουδήποτε αντικειμένου ή function το οποίο τα `handlers` μπορεί να χρησημοποιούν, αυτά λεγόνται εξαρτήσεις(dependencies). Τα Hero funcs μπορούν επίσης να επιστρέψουν οποιαδήποτε τιμή, οποιουδήποτε τύπου, αυτές οι τιμές αποστέλλονται στον πελάτη(client).

> Μπορεί να είχατε ξαναδει "εξαρτήσεις" και "δέσιμο" πριν αλλά ποτέ με υποστήριξη από τον επεξεργαστή κώδικα (code editor), με το Iris πέρνεις πραγματικά ασφαλή "δεσίματα"(bindings) χάρη στο νέο `hero` πακέτο. Αυτή η όλη εκτέλεση(implementation) είναι επίσης η ποιο γρήγορη που έχει επιτευχθεί εως τώρα, η επίδοση είναι πολύ κοντά στα απλά "handlers" και αυτό γιατί το Iris υπολογίζει τα πάντα πριν καν ο server τρέξει!

Παρακάτω θα δείτε μερικά στιγμιότυπα που ετοιμάσαμε για εσάς, ώστε να γίνουν πιο κατανοητά τα παραπάνω:

### 1. Παράμετροι διαδρομής - Ενσωματωμένες Εξαρτήσεις (Built'n Dependencies)

![](https://github.com/kataras/explore/raw/master/iris/hero/hero-1-monokai.png)

### 2. Υπηρεσίες - Στατικές Eξαρτήσεις (Static Dependencies)

![](https://github.com/kataras/explore/raw/master/iris/hero/hero-2-monokai.png)

### 3. Ανά Αίτηση - Δυναμικές Eξαρτήσεις (Dynamic Dependencies)

![](https://github.com/kataras/explore/raw/master/iris/hero/hero-3-monokai.png)

Είναι πραγματικά πολύ εύκολο να καταλάβεις πως δουλεύουν τα `hero funcs` και όταν αρχίσεις να τα χρησιμοποιείς **δεν γυρίζεις ποτέ πίσω**.

Παραδείγματα:

- [Βασικό](_examples/hero/basic/main.go)
- [Επισκόπηση](_examples/hero/overview)

## MVC

Πρέπει να καταλάβεις πως δουλεύει το `hero` πακετό ώστε να να δουλέψεις με το `mvc`, γιατί το `mvc` πακέτο βασίζετε στο `hero` για τις μεθόδους του controller σου, οι ίδιοι κανόνες εφαρμόζονται και εκεί.

Παραδείγματα:

**Αν χρησημοποιούσατε το MVC πριν, διαβάστε προσεχτικά: Το MVC ΠΕΡΙΕΧΕΙ ΟΡΙΣΜΕΝΕΣ ΑΛΛΑΓΕΣ, ΜΠΟΡΕΙΤΕ ΝΑ ΚΑΝΕΤΕ ΠΕΡΙΣΣΟΤΕΡΑ ΑΠΟ'ΤΙ ΠΡΙΝ**

**ΠΑΡΑΚΑΛΩ ΔΙΑΒΑΣΤΕ ΤΑ ΠΑΡΑΔΕΙΓΜΑΤΑ ΠΡΟΣΕΧΤΙΚΑ, ΓΙΑ ΕΣΑΣ ΦΤΙΑΧΤΗΚΑΝ**

Τα παλιά παραδείγματα είναι επίσης εδώ ώστε να μπορείτε να τα συγκρίνετε με τα καινούρια.

| ΤΩΡΑ | ΠΡΙΝ |
| -----------|-------------|
| [Hello world](_examples/mvc/hello-world/main.go) | [ΠΑΛΙΟ Hello world](https://github.com/kataras/iris/blob/v8/_examples/mvc/hello-world/main.go) |
| [Session Controller](_examples/mvc/session-controller/main.go) | [ΠΑΛΙΟ Session Controller](https://github.com/kataras/iris/blob/v8/_examples/mvc/session-controller/main.go) |
| [Overview - Plus Repository and Service layers](_examples/mvc/overview) | [ΠΑΛΙΟ Overview - Plus Repository and Service layers](https://github.com/kataras/iris/tree/v8/_examples/mvc/overview) |
| [Login showcase - Plus Repository and Service layers](_examples/mvc/login) | [ΠΑΛΙΟ Login showcase - Plus Repository and Service layers](https://github.com/kataras/iris/tree/v8/_examples/mvc/login) |
| [Singleton](_examples/mvc/singleton) |  **NEO** |
| [Websocket Controller](_examples/mvc/websocket) |  **NEO** |
| [Vue.js Todo MVC](_examples/tutorial/vuejs-todo-mvc) |  **NEO** |

## context#PostMaxMemory

Αφαίρεση της παλιάς στατικής μεταβλητής `context.DefaultMaxMemory` και αντικατάσταση με ρύθμιση στο configuration `WithPostMaxMemory`.

```go
// WithPostMaxMemory sets the maximum post data size
// that a client can send to the server, this differs
// from the overral request body size which can be modified
// by the `context#SetMaxRequestBodySize` or `iris#LimitRequestBodySize`.
//
// Defaults to 32MB or 32 << 20 if you prefer.
func WithPostMaxMemory(limit int64) Configurator
```

Αν χρησημοποιούσατε την παλιά στατική μεταβλητή θα χρειαστεί να κάνετε μια αλλαγή σε εκείνη τη γραμμή του κώδικα.

```go
import "github.com/kataras/iris"

func main() {
    app := iris.New()
    // [...]

    app.Run(iris.Addr(":8080"), iris.WithPostMaxMemory(10 << 20))
}
```

## context#UploadFormFiles

Νέα μέθοδος για ανέβασμα πολλαπλών αρχείων από τον client, πρέπει να χρησημοποιείται για απλές και συνηθισμένες καταστάσεις, ένα helper είναι μόνο.

```go
// UploadFormFiles uploads any received file(s) from the client
// to the system physical location "destDirectory".
//
// The second optional argument "before" gives caller the chance to
// modify the *miltipart.FileHeader before saving to the disk,
// it can be used to change a file's name based on the current request,
// all FileHeader's options can be changed. You can ignore it if
// you don't need to use this capability before saving a file to the disk.
//
// Note that it doesn't check if request body streamed.
//
// Returns the copied length as int64 and
// a not nil error if at least one new file
// can't be created due to the operating system's permissions or
// http.ErrMissingFile if no file received.
//
// If you want to receive & accept files and manage them manually you can use the `context#FormFile`
// instead and create a copy function that suits your needs, the below is for generic usage.
//
// The default form's memory maximum size is 32MB, it can be changed by the
//  `iris#WithPostMaxMemory` configurator at main configuration passed on `app.Run`'s second argument.
//
// See `FormFile` to a more controlled to receive a file.
func (ctx *context) UploadFormFiles(
        destDirectory string,
        before ...func(string, string),
    ) (int64, error)
```

Το παράδειγμα μπορεί να βρεθεί [εδώ](_examples/http_request/upload-files/main.go).

## context#View

Απλά μια μικρή προσθήκη μια δεύτερης προαιρετικής variadic παράμετρου στη `context#View` μέθοδο για να δέχεται μια μονή τιμή για template binding.
Όταν απλά θέλετε να εμφάνισετε ένα struct value και όχι ζεύγη από κλειδί-τιμή, παλιότερα για να το κάνετε αυτό έπρεπε να δηλώσετε κενό string στη 1η παράμετρο στη `context#ViewData` μέθοδο, το οποίο είναι μια χαρά ειδικά αν δηλώνατε αυτά τα δεδομένα σε προηγούμενο handler της αλυσίδας.

```go
func(ctx iris.Context) {
    ctx.ViewData("", myItem{Name: "iris" })
    ctx.View("item.html")
}
```

Το ίδιο όπως:

```go
func(ctx iris.Context) {
    ctx.View("item.html", myItem{Name: "iris" })
}
```

```html
Item's name: {{.Name}}
```

## context#YAML

Προσθήκη νέας μεθόδου, `context#YAML`, εμφανίζει δομημένο yaml κείμενο από struct value.

```go
// YAML marshals the "v" using the yaml marshaler and renders its result to the client.
func YAML(v interface{}) (int, error)
```

## Session#GetString

Η μέθοδος `sessions/session#GetString` μπορεί πλέον να επιστρέψει τιμή ακόμα και απο τιμή που αποθηκεύτηκαι σαν αριθμός (integer), όπως ακριβώς κάνουν ήδη τα: memstore, η προσωρινή μνήμη του context, οι δυναμικές μεταβλητές του path routing και οι παράμετροι του url query.