# jsonpath

a (partial) implementation in [Go](http://golang.org) based on [Stefan Goener JSON Path](http://goessner.net/articles/JsonPath/)

## Limitations

* No support for subexpressions : `$books[(@.length-1)]`
* No support for filters : `$books[?(@.price > 10)]`
* Strings in brackets must use double quotes : `$["bookstore"]`
* Cannot operate on struct fields

The third limitation comes from using the `text/scanner` package from the standard library.
The last one could be overcome by using reflection.

## JsonPath quick intro

All expressions start `$`.

Examples (supported by the current implementation) :
 * `$` the current object (or array)
 * `$.books` access to the key of an object (or `$["books"]`with bracket syntax)
 * `$.books[1]` access to the index of an array
 * `$.books[1].authors[1].name` chaining of keys and index
 * `$["books"][1]["authors"][1]["name"]` the same with braket syntax
 * `$.books[0,1,3]` union on an array
 * `$["books", "songs", "movies"]` union on an object
 * `$books[1:3]` second and third items of an array
 * `$books[:-2:2]` every two items except the last two of an array
 * `$books[::-1]` all items in reversed order
 * `$..authors` all authors (recursive search)

Checkout the [tests](jsonpath_test.go) for more examples.

## Install

    go get github.com/yalp/jsonpath

## Usage

A jsonpath applies to any JSON decoded data using `interface{}` when decoded with [encoding/json](http://golang.org/pkg/encoding/json/) :

    var bookstore interface{}
    err := json.Unmarshal(data, &bookstore)
    authors, err := jsonpath.Read(bookstore, "$..authors")

A jsonpath expression can be prepared to be reused multiple times :

    allAuthors, err = jsonpath.Prepare("$..authors")
    ...
    var bookstore interface{}
    err := json.Unmarshal(data, &bookstore)
    authors, err := allAuthors(bookstore)

The type of the values returned by the `Read` method or `Prepare` functions depends on the jsonpath expression.
