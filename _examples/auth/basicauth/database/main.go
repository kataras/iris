package main // Look README.md

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/middleware/basicauth"

	_ "github.com/go-sql-driver/mysql" // lint: mysql driver.
)

// User is just an example structure of a user,
// it MUST contain a Username and Password exported fields
// or/and complete the basicauth.User interface.
type User struct {
	ID       int64  `db:"id" json:"id"`
	Username string `db:"username" json:"username"`
	Password string `db:"password" json:"password"`
	Email    string `db:"email" json:"email"`
}

// GetUsername returns the Username field.
func (u User) GetUsername() string {
	return u.Username
}

// GetPassword returns the Password field.
func (u User) GetPassword() string {
	return u.Password
}

func main() {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:3306)/%s?parseTime=true&charset=utf8mb4&collation=utf8mb4_unicode_ci",
		getenv("MYSQL_USER", "user_myapp"),
		getenv("MYSQL_PASSWORD", "dbpassword"),
		getenv("MYSQL_HOST", "localhost"),
		getenv("MYSQL_DATABASE", "myapp"),
	)
	db, err := connect(dsn)
	if err != nil {
		panic(err)
	}

	// Validate a user from database.
	allowFunc := func(ctx iris.Context, username, password string) (interface{}, bool) {
		user, err := db.getUserByUsernameAndPassword(context.Background(), username, password)
		return user, err == nil
	}

	opts := basicauth.Options{
		Realm:        basicauth.DefaultRealm,
		ErrorHandler: basicauth.DefaultErrorHandler,
		Allow:        allowFunc,
	}

	auth := basicauth.New(opts)

	app := iris.New()
	app.Use(auth)
	app.Get("/", index)
	app.Listen(":8080")
}

func index(ctx iris.Context) {
	user, _ := ctx.User().GetRaw()
	// user is a type of main.User
	ctx.JSON(user)
}

func getenv(key string, def string) string {
	v := os.Getenv(key)
	if v == "" {
		return def
	}

	return v
}

type database struct {
	*sql.DB
}

func connect(dsn string) (*database, error) {
	conn, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	err = conn.Ping()
	if err != nil {
		conn.Close()
		return nil, err
	}

	return &database{conn}, nil
}

func (db *database) getUserByUsernameAndPassword(ctx context.Context, username, password string) (User, error) {
	query := fmt.Sprintf("SELECT * FROM %s WHERE %s = ? AND %s = ? LIMIT 1", "users", "username", "password")
	rows, err := db.QueryContext(ctx, query, username, password)
	if err != nil {
		return User{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		return User{}, sql.ErrNoRows
	}

	var user User
	err = rows.Scan(&user.ID, &user.Username, &user.Password, &user.Email)
	return user, err
}
