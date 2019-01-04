package controllers

import "github.com/kataras/iris/mvc"

type HomeController struct{}

func (c *HomeController) Get() mvc.Result {
	return mvc.View{Name: "index.html"}
}

func (c *HomeController) GetAbout() mvc.Result {
	return mvc.View{
		Name: "about.html",
		Data: map[string]interface{}{
			"Title":   "About Page",
			"Message": "Your application description page."},
	}
}

func (c *HomeController) GetContact() mvc.Result {
	return mvc.View{
		Name: "contact.html",
		Data: map[string]interface{}{
			"Title":   "Contact Page",
			"Message": "Your application description page."},
	}
}
