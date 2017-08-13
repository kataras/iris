package controllers

// Index is our index example controller.
type Index struct {
	Controller
}

func (c *Index) Get() {
	c.Tmpl = "index.html"
	c.Data["title"] = "Index page"
	c.Data["message"] = "Hello world!"
}
