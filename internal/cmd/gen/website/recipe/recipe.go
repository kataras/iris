package recipe

import (
	"github.com/kataras/iris/internal/cmd/gen/website/recipe/example"
)

type Recipe struct {
	Branch   string // i.e "master", "v6"...
	Examples []example.Example
}

// NewRecipe accepts the "branch", i.e: "master", "v6", "v7"...
// and returns a new Recipe pointer with its generated and parsed examples.
func NewRecipe(branch string) (*Recipe, error) {
	if branch == "" {
		branch = "master"
	}

	examples, err := example.Parse(branch)
	if err != nil {
		return nil, err
	}

	r := &Recipe{
		Branch:   branch,
		Examples: examples,
	}

	return r, nil
}
