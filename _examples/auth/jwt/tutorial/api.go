package main

import "github.com/kataras/iris/v12"

func loginView(ctx iris.Context) {

}

func login(ctx iris.Context) {

}

func logout(ctx iris.Context) {
	ctx.Logout()

	ctx.Redirect("/", iris.StatusTemporaryRedirect)
}

func createTodo(ctx iris.Context) {

}

func getTodo(ctx iris.Context) {

}
