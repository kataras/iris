package context

import (
	"testing"

	"github.com/kataras/iris/v14/core/dep"
)

func TestRegisterRequestDependencyType_Struct(t *testing.T) {
	type service struct {
		Ctx *Context
	}

	deps := []any{
		func() *Context {
			return &Context{}
		},
		func(s *Context) *service { return &service{Ctx: s} },
		func(s *service) string {
			if s.Ctx == nil {
				return "nil"
			}
			return "not nil"
		},
	}

	container := dep.NewContainer()
	for i, d := range deps {
		err := container.RegisterDependency(d)
		if i == 2 {
			if err == nil {
				t.Fatalf("[%d] expected a not nil error because the Context cannot be registered and resolved as a dependency even if it's provided by the first func", i)
			}
		} else if err != nil {
			t.Fatalf("[%d] %v", i, err)
		}
	}
}
