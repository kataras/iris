package entity

import (
	"fmt"

	"github.com/kataras/iris/v14/context"

	"github.com/kataras/pg"
)

// PageOptions is a simple struct which describes the pagination options.
// Type alias of context.PageOptions.
type PageOptions = context.PageOptions

// ErrNotFound is a type alias of pg.ErrNoRows.
var ErrNotFound = pg.ErrNoRows

// ErrInvalidArgument is a custom error for invalid arguments.
var ErrInvalidArgument = fmt.Errorf("invalid argument")

// ErrAlreadyExists is a custom error for already existing entities.
var ErrAlreadyExists = fmt.Errorf("already exists")
