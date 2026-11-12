package iris_test

import (
	"testing"

	"github.com/iris-contrib/httpexpect/v2"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/core/pkg/netutil"
	"github.com/kataras/iris/v14/httptest"
)

func optionsJSON(t *testing.T, app *iris.Application) *httpexpect.Object {
	t.Helper()

	app.Get("/options", func(ctx iris.Context) {
		ctx.JSON(iris.Map{"options": ctx.Options()})
	})

	e := httptest.New(t, app)
	return e.GET("/options").Expect().Status(httptest.StatusOK).
		JSON().Object().Value("options").Object()
}

// A partial Options literal passed to iris.New must inherit every unset
// field from the defaults instead of silently zeroing them.
func TestOptionsPartialLiteralMergesWithDefaults(t *testing.T) {
	app := iris.New(iris.Options{
		Context: iris.ContextOptions{
			RemoteAddrHeaders: []string{"X-Forwarded-For"},
		},
	})

	obj := optionsJSON(t, app)

	// The custom field is kept.
	obj.Value("remote_addr_headers").Array().ConsistsOf("X-Forwarded-For")
	// The unset fields inherit the defaults.
	obj.HasValue("charset", "utf-8")
	obj.HasValue("post_max_memory", 32<<20)
	obj.HasValue("time_format", "Mon, 02 Jan 2006 15:04:05 GMT")
	obj.Value("remote_addr_private_subnets").Array().Length().IsEqual(6)
}

// The zero Options literal behaves exactly like no options at all.
func TestOptionsZeroValueEqualsDefaults(t *testing.T) {
	app := iris.New(iris.Options{})

	obj := optionsJSON(t, app)
	obj.HasValue("charset", "utf-8")
	obj.HasValue("post_max_memory", 32<<20)
}

// A non-nil empty slice is an explicit "none" and must not be re-defaulted.
func TestOptionsExplicitEmptySliceKept(t *testing.T) {
	app := iris.New(iris.Options{
		Context: iris.ContextOptions{
			RemoteAddrPrivateSubnets: []netutil.IPRange{},
		},
	})

	obj := optionsJSON(t, app)
	// omitempty: an empty subnets slice never renders, while the defaults
	// (six ranges) would.
	obj.NotContainsKey("remote_addr_private_subnets")
}
