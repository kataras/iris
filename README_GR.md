# Iris <a href="README.md"> <img width="20px" src="https://iris-go.com/images/flag-unitedkingdom.svg?v=10" /></a> <a href="README_ZH.md"><img width="20px" src="https://iris-go.com/images/flag-china.svg?v=10" /></a> <a href="README_ES.md"><img width="20px" src="https://iris-go.com/images/flag-spain.png" /></a> <a href="README_KO.md"><img width="20px" src="https://iris-go.com/images/flag-south-korea.svg" /></a> <a href="README_FA.md"><img width="20px" src="https://iris-go.com/images/flag-iran.svg" /></a> <a href="README_RU.md"><img width="20px" src="https://iris-go.com/images/flag-russia.svg" /></a>

[![build status](https://img.shields.io/travis/kataras/iris/master.svg?style=for-the-badge&logo=travis)](https://travis-ci.org/kataras/iris) [![report card](https://img.shields.io/badge/report%20card-a%2B-ff3333.svg?style=for-the-badge)](https://goreportcard.com/report/github.com/kataras/iris)<!--[![godocs](https://img.shields.io/badge/go-%20docs-488AC7.svg?style=for-the-badge)](https://godoc.org/github.com/kataras/iris)--> [![view examples](https://img.shields.io/badge/learn%20by-examples-0077b3.svg?style=for-the-badge)](https://github.com/kataras/iris/tree/master/_examples) [![chat](https://img.shields.io/gitter/room/iris_go/community.svg?color=blue&logo=gitter&style=for-the-badge)](https://gitter.im/iris_go/community) [![release](https://img.shields.io/badge/release%20-v11.2-0077b3.svg?style=for-the-badge)](https://github.com/kataras/iris/releases)

Το Iris είναι ένα γρήγορο, απλό αλλά και πλήρως λειτουργικό και πολύ αποδοτικό web framework για τη Go γλώσσα προγραμματισμού. Παρέχει ένα εκφραστικό και εύχρηστο υπόβαθρο για την επόμενη ιστοσελίδα σας.

Μάθετε τι [λένε οι άλλοι για το Iris](https://iris-go.com/testimonials/) και δώστε ένα **αστεράκι** στο GitHub.

[![](https://media.giphy.com/media/j5WLmtvwn98VPrm7li/giphy.gif)](https://iris-go.com/testimonials/)

## Μαθαίνοντας το Iris

<details>
<summary>Γρήγορο ξεκίνημα</summary>

```sh
# υποθέτοντας ότι ο παρακάτω κώδικας
# βρίσκεται στο example.go αρχείο
#
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
# τρέξτε το example.go και
# επισκεφτείτε την σελίδα http://localhost:8080/ping
# στο πρόγραμμα περιήγησης σας
#
$ go run example.go
```

> Η δρομολόγηση τροφοδοτείται από το [muxie](https://github.com/kataras/muxie), το πιο ισχυρό και ταχύτερο λογισμικό βασισμένο σε trie αλγόριθμο που γράφτηκε σε Go.

</details>

Το Iris περιέχει εκτενείς και λεπτομερείς **[wiki](https://github.com/kataras/iris/wiki)** καθιστώντας το εύκολο στην εκμάθηση.

Για λεπτομερέστερη τεχνική τεκμηρίωση μπορείτε να κατευθυνθείτε προς τα [godocs](https://godoc.org/github.com/kataras/iris) μας. Και για εκτελέσιμο κώδικα μπορείτε πάντα να επισκέπτεστε τα [παραδείγματα](_examples/).

### Σας αρέσει να διαβάζετε ενώ ταξιδεύετε;

Μπορείτε να [ζητήσετε](https://bit.ly/iris-req-book) σήμερα την PDF έκδοση και την online πρόσβαση στο Ηλεκτρονικό μας **Βιβλίο(E-Book)** και να συμμετάσχετε στην ανάπτυξη του Iris.

[![https://iris-go.com/images/iris-book-overview.png](https://iris-go.com/images/iris-book-overview.png)](https://bit.ly/iris-req-book)

[![follow author](https://img.shields.io/twitter/follow/makismaropoulos.svg?style=for-the-badge)](https://twitter.com/intent/follow?screen_name=makismaropoulos)

## Συνεισφορά

Θα θέλαμε να δούμε τη συμβολή σας στο Iris Web Framework! Για περισσότερες πληροφορίες σχετικά με το πως μπορείτε να συμβάλετε, δείτε το [CONTRIBUTING.md](CONTRIBUTING.md) αρχείο.

[Κατάλογος όλων των συνεισφορών](https://github.com/kataras/iris/graphs/contributors).

## Αδυναμίες Ασφάλειας

Εάν εντοπίσετε κάποια αδυναμία ασφαλείας του Iris, στείλτε ένα μήνυμα ηλεκτρονικού ταχυδρομείου στο [iris-go@outlook.com](mailto:iris-go@outlook.com). Όλες οι τυχών αδυναμίες ασφαλείας θα αντιμετωπιστούν άμεσα.

## Άδεια Χρήσης

Το όνομα "Iris" εμπνεύστηκε από την ελληνική μυθολογία, από την θεά Ίριδα.

Το Iris Web Framework είναι δωρεάν λογισμικό ανοιχτού λογισμικού με άδεια χρήσης [3-Clause BSD](LICENSE).
