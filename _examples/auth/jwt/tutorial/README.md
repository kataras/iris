# Iris JWT Tutorial

This example show how to use JWT with domain-driven design pattern with Iris. There is also a simple Go client which describes how you can use Go to authorize a user and use the server's API.

## Run the server

```sh
$ go run main.go
```

## Authenticate, get the token

```sh
$ curl --location --request POST 'http://localhost:8080/signin' \
--header 'Content-Type: application/x-www-form-urlencoded' \
--data-urlencode 'username=admin' \
--data-urlencode 'password=admin'

> $token
```

## Get all TODOs for this User

```sh
$ curl --location --request GET 'http://localhost:8080/todos' \
--header 'Authorization: Bearer $token'

> $todos
```

## Get a specific User's TODO 

```sh
$ curl --location --request GET 'http://localhost:8080/todos/$id' \
--header 'Authorization: Bearer $token'

> $todo
```

## Get all TODOs for all Users (admin role)

```sh
$ curl --location --request GET 'http://localhost:8080/admin/todos' \
--header 'Authorization: Bearer $token'

> $todos
```

## Create a new TODO

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
