```go
package main

import (
	"fmt"
	"strings"

	"github.com/kataras/iris/v12/macro"
)

func main() {
	path := NewRouteBuilder().
		Path("/user").
		String("name", "prefix(ma)", "suffix(kis)").
		Int("age").
		Path("/friends").
		Wildcard("rest").
		Build()

	fmt.Println(path)
}

type RouteBuilder struct {
	path string
}

func NewRouteBuilder() *RouteBuilder {
	return &RouteBuilder{
		path: "/",
	}
}

func (r *RouteBuilder) Path(path string) *RouteBuilder {
	if path[0] != '/' {
		path = "/" + path
	}

	r.path = strings.TrimSuffix(r.path, "/") + path
	return r
}

type StaticPathBuilder interface {
	Path(path string) *RouteBuilder
}

func (r *RouteBuilder) Param(param ParamBuilder) *RouteBuilder { // StaticPathBuilder {
	path := "" // keep it here, a single call to r.Path must be done.
	if len(r.path) == 0 || r.path[len(r.path)-1] != '/' {
		path += "/" // if for some reason no prior Path("/") was called for delimeter between path parameter.
	}

	path += fmt.Sprintf("{%s:%s", param.GetName(), param.GetParamType().Indent())
	if funcs := param.GetFuncs(); len(funcs) > 0 {
		path += fmt.Sprintf(" %s", strings.Join(funcs, " "))
	}
	path += "}"

	return r.Path(path)
}

func (r *RouteBuilder) String(name string, funcs ...string) *RouteBuilder {
	return r.Param(Param(macro.String, name, funcs...))
}

func (r *RouteBuilder) Int(name string, funcs ...string) *RouteBuilder {
	return r.Param(Param(macro.Int, name, funcs...))
}

func (r *RouteBuilder) Int8(name string, funcs ...string) *RouteBuilder {
	return r.Param(Param(macro.Int8, name, funcs...))
}

func (r *RouteBuilder) Int16(name string, funcs ...string) *RouteBuilder {
	return r.Param(Param(macro.Int16, name, funcs...))
}

func (r *RouteBuilder) Int32(name string, funcs ...string) *RouteBuilder {
	return r.Param(Param(macro.Int32, name, funcs...))
}

func (r *RouteBuilder) Int64(name string, funcs ...string) *RouteBuilder {
	return r.Param(Param(macro.Int64, name, funcs...))
}

func (r *RouteBuilder) Uint(name string, funcs ...string) *RouteBuilder {
	return r.Param(Param(macro.Uint, name, funcs...))
}

func (r *RouteBuilder) Uint8(name string, funcs ...string) *RouteBuilder {
	return r.Param(Param(macro.Uint8, name, funcs...))
}

func (r *RouteBuilder) Uint16(name string, funcs ...string) *RouteBuilder {
	return r.Param(Param(macro.Uint16, name, funcs...))
}

func (r *RouteBuilder) Uint32(name string, funcs ...string) *RouteBuilder {
	return r.Param(Param(macro.Uint32, name, funcs...))
}

func (r *RouteBuilder) Uint64(name string, funcs ...string) *RouteBuilder {
	return r.Param(Param(macro.Uint64, name, funcs...))
}

func (r *RouteBuilder) Bool(name string, funcs ...string) *RouteBuilder {
	return r.Param(Param(macro.Bool, name, funcs...))
}

func (r *RouteBuilder) Alphabetical(name string, funcs ...string) *RouteBuilder {
	return r.Param(Param(macro.Alphabetical, name, funcs...))
}

func (r *RouteBuilder) File(name string, funcs ...string) *RouteBuilder {
	return r.Param(Param(macro.File, name, funcs...))
}

func (r *RouteBuilder) Wildcard(name string, funcs ...string) *RouteBuilder {
	return r.Param(Param(macro.Path, name, funcs...))
}

func (r *RouteBuilder) UUID(name string, funcs ...string) *RouteBuilder {
	return r.Param(Param(macro.UUID, name, funcs...))
}

func (r *RouteBuilder) Mail(name string, funcs ...string) *RouteBuilder {
	return r.Param(Param(macro.Mail, name, funcs...))
}

func (r *RouteBuilder) Email(name string, funcs ...string) *RouteBuilder {
	return r.Param(Param(macro.Email, name, funcs...))
}

func (r *RouteBuilder) Date(name string, funcs ...string) *RouteBuilder {
	return r.Param(Param(macro.Date, name, funcs...))
}

func (r *RouteBuilder) Weekday(name string, funcs ...string) *RouteBuilder {
	return r.Param(Param(macro.Weekday, name, funcs...))
}

func (r *RouteBuilder) Build() string {
	return r.path
}

type ParamBuilder interface {
	GetName() string
	GetFuncs() []string
	GetParamType() *macro.Macro
}

type pathParam struct {
	Name      string
	Funcs     []string
	ParamType *macro.Macro
}

var _ ParamBuilder = (*pathParam)(nil)

func Param(paramType *macro.Macro, name string, funcs ...string) ParamBuilder {
	return &pathParam{
		Name:      name,
		ParamType: paramType,
		Funcs:     funcs,
	}
}

func (p *pathParam) GetName() string {
	return p.Name
}

func (p *pathParam) GetParamType() *macro.Macro {
	return p.ParamType
}

func (p *pathParam) GetFuncs() []string {
	return p.Funcs
}
```
