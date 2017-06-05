package examples

// Category defines the category of which will contain examples.
type Category struct {
	Name     string // i.e "Beginner", "Intermediate", "Advanced", first upper.
	Examples []Example
}

// Example defines the example link.
type Example struct {
	Name       string // i.e: Hello World
	DataSource string // i.e: https://raw.githubusercontent.com/kataras/iris/master/_examples/beginner/hello-world.go
}
