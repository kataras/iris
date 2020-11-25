# Iris, MySQL, Groupcache & Docker Example

## 📘 Endpoints

| Method | Path                | Description            | URL Parameters | Body                       | Auth Required |
|--------|---------------------|------------------------|--------------- |----------------------------|---------------|
| ANY    | /token              | Prints a new JWT Token | -              | -                          | -             |
| GET    | /category           | Lists a set of Categories    | offset, limit, order | -                    | Token         |
| POST   | /category           | Creates a Category      | -              | JSON [Full Category](migration/api_category/create_category.json)              | Token      |
| PUT    | /category           | Fully-Updates a Category | -              | JSON [Full Category](migration/api_category/update_category.json)              | Token      |
| PATCH  | /category/{id}      | Partially-Updates a Category | -              | JSON [Partial Category](migration/api_category/update_partial_category.json)              | Token      |
| GET    | /category/{id}      | Prints a Category         | -              | -      | Token      |
| DELETE | /category/{id}      | Deletes a Category      | -              | -      | Token      |
| GET    | /category/{id}/products | Lists all Products from a Category      | offset, limit, order | -      | Token      |
| POST   | /category/{id}/products | (Batch) Assigns one or more Products to a Category      | -              | JSON [Products](migration/api_category/insert_products_category.json)      | Token      |
| GET    | /product           | Lists a set of Products (cache)     | offset, limit, order | -                    | Token      |
| POST   | /product           | Creates a Product      | -              | JSON [Full Product](migration/api_product/create_product.json)              | Token      |
| PUT    | /product           | Fully-Updates a Product | -              | JSON [Full Product](migration/api_product/update_product.json)              | Token      |
| PATCH  | /product/{id}      | Partially-Updates a Product | -              | JSON [Partial Product](migration/api_product/update_partial_product.json)              | Token      |
| GET    | /product/{id}      | Prints a Product (cache)         | -              | -      | Token      |
| DELETE | /product/{id}      | Deletes a Product        | -              | -      | Token      |



## 📑 Responses

* **Content-Type** of `"application/json;charset=utf-8"`, snake_case naming (identical to the database columns)
* **Status Codes**
    * 500 for server(db) errors,
    * 422 for validation errors, e.g.
    ```json
    {
        "code": 422,
        "message": "required fields are missing",
        "timestamp": 1589306271
    }
    ```
    * 400 for malformed syntax, e.g.
    ```json
    {
        "code": 400,
        "message": "json: cannot unmarshal number -2 into Go struct field Category.position of type uint64",
        "timestamp": 1589306325
    }
    ```
    ```json
    {
        "code": 400,
        "message": "json: unknown field \"field_not_exists\"",
        "timestamp": 1589306367
    }
    ```
    * 404 for entity not found, e.g.
    ```json
    {
        "code": 404,
        "message": "entity does not exist",
        "timestamp": 1589306199
    }
    ```
    * 304 for unaffected UPDATE or DELETE,
    * 201 for CREATE with the last inserted ID,
    * 200 for GET, UPDATE and DELETE

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

```sql
CREATE DATABASE IF NOT EXISTS myapp DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

USE myapp;

SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

DROP TABLE IF EXISTS categories;
CREATE TABLE categories  (
  id int(11) NOT NULL AUTO_INCREMENT,
  title varchar(255) NOT NULL,
  position int(11) NOT NULL,
  image_url varchar(255) NOT NULL,
  created_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id)
);

DROP TABLE IF EXISTS products;
CREATE TABLE products  (
  id int(11) NOT NULL AUTO_INCREMENT,
  category_id int,
  title varchar(255) NOT NULL,
  image_url varchar(255) NOT NULL,
  price decimal(10,2) NOT NULL,
  description text NOT NULL,
  created_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  FOREIGN KEY (category_id) REFERENCES categories(id)
);

SET FOREIGN_KEY_CHECKS = 1;
```

### Requests

Some request bodies can be found at: [migration/api_category](migration/api_category) and [migration/api_product](migration/api_product). **However** I've provided a [postman.json](migration/myapp_postman.json) Collection that you can import to your [POSTMAN](https://learning.postman.com/docs/postman/collections/importing-and-exporting-data/#collections) and start playing with the API.

All write-access endpoints are "protected" via JWT, a client should "verify" itself. You'll need to manually take the **token** from the `http://localhost:8080/token` and put it on url parameter `?token=$token` or to the `Authentication: Bearer $token` request header.

### Unit or End-To-End Testing?

Testing is important. The code is written in a way that testing should be trivial (Pseudo/memory Database or SQLite local file could be integrated as well, for end-to-end tests a Docker image with MySQL and fire tests against that server). However, there is [nothing(?)](service/category_service_test.go) to see here.

## Packages

- https://github.com/kataras/jwt (JWT parsing)
- https://github.com/go-sql-driver/mysql (Go Driver for MySQL)
- https://github.com/DATA-DOG/go-sqlmock (Testing DB see [service/category_service_test.go](service/category_service_test.go))
- https://github.com/kataras/iris (HTTP)
- https://github.com/mailgun/groupcache (Caching)
