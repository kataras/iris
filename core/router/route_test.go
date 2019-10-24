// white-box testing

package router

import (
	"testing"

	"github.com/kataras/iris/v12/macro"
)

func TestRouteStaticPath(t *testing.T) {
	tests := []struct {
		tmpl   string
		static string
	}{
		{
			tmpl:   "/files/{file:path}",
			static: "/files",
		},
		{
			tmpl:   "/path",
			static: "/path",
		},
		{
			tmpl:   "/path/segment",
			static: "/path/segment",
		},
		{
			tmpl:   "/path/segment/{n:int}",
			static: "/path/segment",
		},
		{
			tmpl:   "/path/{n:uint64}/{n:int}",
			static: "/path",
		},
		{
			tmpl:   "/path/{n:uint64}/static",
			static: "/path",
		},
		{
			tmpl:   "/{name}",
			static: "/",
		},
		{
			tmpl:   "/",
			static: "/",
		},
	}

	for i, tt := range tests {
		route := Route{tmpl: macro.Template{Src: tt.tmpl}}
		if expected, got := tt.static, route.StaticPath(); expected != got {
			t.Fatalf("[%d:%s] expected static path to be: '%s' but got: '%s'", i, tt.tmpl, expected, got)
		}
	}
}
