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