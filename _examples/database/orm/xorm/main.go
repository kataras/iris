// Package main shows how an orm can be used within your web app
// it just inserts a column and select the first.
package main

import (
	"time"

	"github.com/kataras/iris/v12"

	_ "github.com/mattn/go-sqlite3"
	"xorm.io/xorm"
)

/*
	go get -u github.com/mattn/go-sqlite3
	go get -u xorm.io/xorm

	If you're on win64 and you can't install go-sqlite3:
		1. Download: https://sourceforge.net/projects/mingw-w64/files/latest/download
		2. Select "x86_x64" and "posix"
		3. Add C:\Program Files\mingw-w64\x86_64-7.1.0-posix-seh-rt_v5-rev1\mingw64\bin
		to your PATH env variable.

	Docs: https://gitea.com/xorm/xorm
*/

// User is our user table structure.
type User struct {
	ID        int64  // auto-increment by-default by xorm
	Version   string `xorm:"varchar(200)"`
	Salt      string
	Username  string
	Password  string    `xorm:"varchar(200)"`
	Languages string    `xorm:"varchar(200)"`
	CreatedAt time.Time `xorm:"created"`
	UpdatedAt time.Time `xorm:"updated"`
}

func main() {
	app := iris.New()

	orm, err := xorm.NewEngine("sqlite3", "./test.db")
	if err != nil {
		app.Logger().Fatalf("orm failed to initialized: %v", err)
	}
	iris.RegisterOnInterrupt(func() {
		orm.Close()
	})

	if err = orm.Sync2(new(User)); err != nil {
		app.Logger().Fatalf("orm failed to initialized User table: %v", err)
	}

	app.Get("/insert", func(ctx iris.Context) {
		user := &User{Username: "kataras", Salt: "hash---", Password: "hashed", CreatedAt: time.Now(), UpdatedAt: time.Now()}
		orm.Insert(user)

		ctx.Writef("user inserted: %#v", user)
	})

	app.Get("/get", func(ctx iris.Context) {
		user := User{ID: 1}
		found, err := orm.Get(&user) // fetch user with ID.
		if err != nil {
			ctx.StopWithError(iris.StatusInternalServerError, err)
			return
		}

		if !found {
			ctx.StopWithText(iris.StatusNotFound, "User with ID: %d not found", user.ID)
			return
		}

		ctx.Writef("User Found: %#v", user)
	})

	// http://localhost:8080/insert
	// http://localhost:8080/get
	app.Listen(":8080")
}
