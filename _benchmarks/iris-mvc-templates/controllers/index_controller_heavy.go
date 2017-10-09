package controllers

import "github.com/kataras/iris/mvc"

type IndexControllerHeavy struct{ mvc.C }

func (c *IndexControllerHeavy) Get() mvc.View {
	return mvc.View{
		Name: "index.html",
		Data: map[string]interface{}{
			"Title": "Home Page",
		},
	}
}
