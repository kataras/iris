# Iris JWT Tutorial

```sh
$ go run main.go
```

```sh
$ curl --location --request POST 'http://localhost:8080/signin' \
--header 'Content-Type: application/x-www-form-urlencoded' \
--data-urlencode 'username=admin' \
--data-urlencode 'password=admin'

> $token
```

```sh
$ curl --location --request GET 'http://localhost:8080/todos' \
--header 'Authorization: Bearer $token'

> $todos
```

```sh
$ curl --location --request GET 'http://localhost:8080/todos/$id' \
--header 'Authorization: Bearer $token'

> $todo
```

```sh
$ curl --location --request GET 'http://localhost:8080/admin/todos' \
--header 'Authorization: Bearer $token'

> $todos
```

```sh
$ curl --location --request POST 'http://localhost:8080/todos' \
--header 'Authorization: Bearer $token' \
--header 'Content-Type: application/json' \
--data-raw '{
    "title": "test titlte",
    "body": "test body"
}'

> Status Created
> $todo
```

TODO: write the article on https://medium.com/@kataras, https://dev.to/kataras and linkedin first.
