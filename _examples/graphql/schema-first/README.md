# outerbanks-api

A graphql api where we can store and get information on characters in Outerbanks.

> This example is an updated version (**2023**) of outerbanks-api and it is based on: https://www.apollographql.com/blog/graphql/golang/using-graphql-with-golang.

![](https://www.iris-go.com/images/graphql_playground.png)

## Getting Started

```sh
$ go install github.com/99designs/gqlgen@latest
```

Add `gqlgen` to your project's `tools.go` file

```sh
$ printf '// +build tools\npackage tools\nimport _ "github.com/99designs/gqlgen"' | gofmt > tools.go
$ go get github.com/kataras/iris/v12@latest
$ go mod tidy -compat=1.20
```

Start the graphql server

```
$ go run .
```

## Mutation

Open http://localhost:8080

On the editor panel paste:

```graphql
mutation upsertCharacter($input:CharacterInput!){
  upsertCharacter(input:$input) {
  	name
    id
  }
}
```

And in the variables panel below, paste:

```json
{
  "input":{
   	"name": "kataras",
    "cliqueType": "POGUES"
  }
}
```

Hit Ctrl+Enter to apply the mutation.


## Query

Query:

```graphql
query character($id:ID!) {
  character(id:$id) {
    id
    name
  }
}
```

Variables:

```json
{
 "id":1
}
```

## Re-generate code

```sh
$ cd graph
$ rm -f graph/schema.resolvers.go
$ touch schema.graphql # make your updates here
$ gqlgen generate
```
