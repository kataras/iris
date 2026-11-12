package versioning_test

import (
	"testing"
	"time"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/httptest"
	"github.com/kataras/iris/v14/middleware/versioning"
)

func TestHandler(t *testing.T) {
	app := iris.New()
	opts := versioning.Options{
		Default: "1.0.0",
		Versions: []versioning.Version{
			{
				Range:      ">=1.0.0 <2.0.0",
				Path:       "/v1",
				Deprecated: true,
			},
			{
				Range: ">=2.0.0",
				Path:  "/v2",
			},
		},
		Deprecation: versioning.DeprecationOptions{
			WarnMessage:     "deprecated, see <this link>",
			DeprecationDate: time.Now().UTC(),
			DeprecationInfo: "a bigger version is available, see <this link> for more information",
		},
	}
	app.UseRouter(versioning.New(opts))

	writeVesion := func(ctx iris.Context) {
		ctx.Writef("%s: %s", ctx.Path(), versioning.Get(ctx))
	}

	v1 := app.Router("/v1")
	v1.Get("/", writeVesion)

	v2 := app.Router("/v2")
	v2.Get("/", writeVesion)

	e := httptest.New(t, app)

	ex := e.GET("/").WithHeader(versioning.AcceptVersionHeaderKey, "1.0.0").Expect()
	ex.Status(iris.StatusOK).Body().IsEqual("/v1: 1.0.0")
	ex.Header(versioning.APIWarnHeader).IsEqual(opts.Deprecation.WarnMessage)
	expectedDateStr := opts.Deprecation.DeprecationDate.Format(context.DefaultTimeFormat)
	ex.Header(versioning.APIDeprecationDateHeader).IsEqual(expectedDateStr)

	e.GET("/").WithHeader(versioning.AcceptVersionHeaderKey, "2.0").Expect().Body().IsEqual("/v2: 2.0.0") // test also the no patch version.
	e.GET("/").WithHeader(versioning.AcceptVersionHeaderKey, "2").Expect().Body().IsEqual("/v2: 2.0.0")   // test also the no minor and patch version.
}
