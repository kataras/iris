package example

// Example defines the example link.
type Example struct {
	Name       string    // i.e: Hello World
	DataSource string    // i.e: https://raw.githubusercontent.com/iris-contrib/examples/master/hello-world.go
	Children   []Example // if has children the data source is not a source file, it's just a folder, its the template's H2 tag.
	// needed for the raw templates, we can do a simple func but lets keep it simple, it's a small template file.
	HasChildren    bool
	HasNotChildren bool
}
