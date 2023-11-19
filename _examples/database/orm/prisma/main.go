//go:generate go run github.com/steebchen/prisma-client-go db push

package main

import (
	"context"
	"demo/db"
	"net/http"
	"strconv"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/middleware/logger"
	"github.com/kataras/iris/v12/middleware/recover"
)

func main() {
	app := iris.Default()

	client := db.NewClient()
	if err := client.Prisma.Connect(); err != nil {
		app.Logger().Fatalf("unable to connect to database: %v", err)
	}

	defer func() {
		if err := client.Prisma.Disconnect(); err != nil {
			app.Logger().Fatal(err)
		}
	}()

	iris.RegisterOnInterrupt(func() {
		client.Prisma.Disconnect()
	})

	app.Use(logger.New())
	app.Use(recover.New())

	app.Post("/tasks", func(ctx iris.Context) {
		var task db.TaskModel
		if err := ctx.ReadJSON(&task); err != nil {
			app.Logger().Error("Bind: ", err)
			ctx.StopWithText(iris.StatusBadRequest, "Bind: "+err.Error())
		}
		var text *string
		if newText, ok := task.Text(); ok {
			text = &newText
		}
		var completed *bool
		if newCompleted, ok := task.Completed(); ok {
			completed = &newCompleted
		}
		newTask, err := client.Task.CreateOne(
			db.Task.Text.SetIfPresent(text),
			db.Task.Completed.SetIfPresent(completed),
		).Exec(context.Background())
		if err != nil {
			ctx.StopWithText(iris.StatusBadRequest, err.Error())
		}
		ctx.StopWithJSON(iris.StatusOK, newTask)
	})

	app.Get("/tasks", func(ctx iris.Context) {
		tasks, err := client.Task.FindMany().OrderBy(
			db.Task.ID.Order(db.ASC),
		).Exec(context.Background())
		if err != nil {
			ctx.StopWithText(iris.StatusBadRequest, err.Error())
		}
		ctx.StopWithJSON(iris.StatusOK, tasks)
	})

	app.Post("/tasks/:id", func(ctx iris.Context) {
		var task db.TaskModel
		if err := ctx.ReadJSON(&task); err != nil {
			app.Logger().Error("Bind: ", err)
			ctx.StopWithText(http.StatusBadRequest, "Bind: "+err.Error())
		}
		var text *string
		if newText, ok := task.Text(); ok {
			text = &newText
		}
		var completed *bool
		if newCompleted, ok := task.Completed(); ok {
			completed = &newCompleted
		}
		newTask, err := client.Task.FindUnique(
			db.Task.ID.Equals(task.ID),
		).Update(
			db.Task.Text.SetIfPresent(text),
			db.Task.Completed.SetIfPresent(completed),
		).Exec(context.Background())
		if err != nil {
			ctx.StopWithText(iris.StatusBadRequest, err.Error())
		}
		ctx.StopWithJSON(iris.StatusOK, newTask)
	})

	app.Delete("/tasks/:id", func(ctx iris.Context) {
		id, err := strconv.Atoi(ctx.Params().Get("id"))
		if err != nil {
			ctx.StopWithText(iris.StatusBadRequest, err.Error())
		}
		task, err := client.Task.FindUnique(
			db.Task.ID.Equals(id),
		).Delete().Exec(context.Background())
		if err != nil {
			ctx.StopWithText(iris.StatusNotFound, err.Error())
		}
		ctx.StopWithJSON(iris.StatusOK, task)
	})
	app.Get("/tasks/:id", func(ctx iris.Context) {
		id, err := strconv.Atoi(ctx.Params().Get("id"))
		if err != nil {
			ctx.StopWithText(iris.StatusBadRequest, err.Error())
		}
		task, err := client.Task.FindUnique(
			db.Task.ID.Equals(id),
		).Exec(context.Background())
		if err != nil {
			ctx.StopWithText(iris.StatusNotFound, err.Error())
		}
		ctx.StopWithJSON(http.StatusOK, task)
	})

	app.Listen(":8080")
}
