# BasicAuth + MySQL & Docker Example

## ⚡ Get Started

Download the folder.

### Install (Docker)

Install [Docker](https://www.docker.com/) and execute the command below

```sh
$ docker-compose up --build
```

### Install (Manually)

Run `go build` or `go run main.go` and read below.

#### MySQL

Environment variables:

```sh
MYSQL_USER=user_myapp
MYSQL_PASSWORD=dbpassword
MYSQL_HOST=localhost
MYSQL_DATABASE=myapp
```

Download the schema from [migration/db.sql](migration/db.sql) and execute it against your MySQL server instance.

<http://localhost:8080>

```sh
username: admin
password: admin
```

```sh
username: iris
password: iris_password
```

The example does not contain code to add a user to the database, as this is out of the scope of this middleware. More features can be implemented by end-developers.
