package main

import (
	"net/http"
	"os"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/kataras/iris"
	_ "github.com/mattn/go-sqlite3"
)

type User struct {
	gorm.Model
	Salt      string `gorm:"type:varchar(255)" json:"salt"`
	Username  string `gorm:"type:varchar(32)" json:"username"`
	Password  string `gorm:"type:varchar(200);column:password" json:"-"`
	Languages string `gorm:"type:varchar(200);column:languages" json:"languages"`
}

func (u User) TableName() string {
	return "gorm_user"
}

type UserSerializer struct {
	ID        uint      `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Salt      string    `json:"salt"`
	UserName  string    `json:"user_name"`
	Password  string    `json:"-"`
	Languages string    `json:"languages"`
}

func (self User) Serializer() UserSerializer {
	return UserSerializer{
		ID:        self.ID,
		CreatedAt: self.CreatedAt.Truncate(time.Second),
		UpdatedAt: self.UpdatedAt.Truncate(time.Second),
		Salt:      self.Salt,
		Password:  self.Password,
		Languages: self.Languages,
		UserName:  self.Username,
	}
}

func main() {
	app := iris.Default()
	db, err := gorm.Open("sqlite3", "test.db")
	db.LogMode(true) // show SQL logger
	if err != nil {
		app.Logger().Fatalf("connect to sqlite3 failed")
		return
	}
	iris.RegisterOnInterrupt(func() {
		defer db.Close()
	})

	if os.Getenv("ENV") != "" {
		db.DropTableIfExists(&User{}) // drop table
	}
	db.AutoMigrate(&User{}) // create table: // AutoMigrate run auto migration for given models, will only add missing fields, won't delete/change current data

	app.Post("/post_user", func(context iris.Context) {
		var user User
		user = User{
			Username:  "gorm",
			Salt:      "hash---",
			Password:  "admin",
			Languages: "gorm",
		}
		if err := db.FirstOrCreate(&user); err == nil {
			app.Logger().Fatalf("created one record failed: %s", err.Error)
			context.JSON(iris.Map{
				"code":  http.StatusBadRequest,
				"error": err.Error,
			})
			return
		}
		context.JSON(
			iris.Map{
				"code": http.StatusOK,
				"data": user.Serializer(),
			})
	})

	app.Get("/get_user/{id:uint}", func(context iris.Context) {
		var user User
		id, _ := context.Params().GetUint("id")
		app.Logger().Println(id)
		if err := db.Where("id = ?", int(id)).First(&user).Error; err != nil {
			app.Logger().Fatalf("find one record failed: %t", err == nil)
			context.JSON(iris.Map{
				"code":  http.StatusBadRequest,
				"error": err.Error,
			})
			return
		}
		context.JSON(iris.Map{
			"code": http.StatusOK,
			"data": user.Serializer(),
		})
	})

	app.Delete("/delete_user/{id:uint}", func(context iris.Context) {
		id, _ := context.Params().GetUint("id")
		if id == 0 {
			context.JSON(iris.Map{
				"code":   http.StatusOK,
				"detail": "query param id should not be nil",
			})
			return
		}
		var user User
		if err := db.Where("id = ?", id).First(&user).Error; err != nil {
			app.Logger().Fatalf("record not found")
			context.JSON(iris.Map{
				"code":   http.StatusOK,
				"detail": err.Error,
			})
			return
		}
		db.Delete(&user)
		context.JSON(iris.Map{
			"code": http.StatusOK,
			"data": user.Serializer(),
		})
	})

	app.Patch("/patch_user/{id:uint}", func(context iris.Context) {
		id, _ := context.Params().GetUint("id")
		if id == 0 {
			context.JSON(iris.Map{
				"code":   http.StatusOK,
				"detail": "query param id should not be nil",
			})
			return
		}
		var user User
		tx := db.Begin()
		if err := tx.Where("id = ?", id).First(&user).Error; err != nil {
			app.Logger().Fatalf("record not found")
			context.JSON(iris.Map{
				"code":   http.StatusOK,
				"detail": err.Error,
			})
			return
		}

		var body patchParam
		context.ReadJSON(&body)
		app.Logger().Println(body)
		if err := tx.Model(&user).Updates(map[string]interface{}{"username": body.Data.UserName, "password": body.Data.Password}).Error; err != nil {
			app.Logger().Fatalf("update record failed")
			tx.Rollback()
			context.JSON(iris.Map{
				"code":  http.StatusBadRequest,
				"error": err.Error,
			})
			return
		}
		tx.Commit()
		context.JSON(iris.Map{
			"code": http.StatusOK,
			"data": user.Serializer(),
		})
	})
	app.Run(iris.Addr(":8080"), iris.WithoutServerError(iris.ErrServerClosed))
}

type patchParam struct {
	Data struct {
		UserName string `json:"user_name" form:"user_name"`
		Password string `json:"password" form:"password"`
	} `json:"data"`
}
