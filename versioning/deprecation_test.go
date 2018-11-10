package versioning_test

import (
	"testing"
	"time"

	"github.com/kataras/iris"
	"github.com/kataras/iris/httptest"
	"github.com/kataras/iris/versioning"
)

func TestDeprecated(t *testing.T) {
	app := iris.New()

	writeVesion := func(ctx iris.Context) {
		ctx.WriteString(versioning.GetVersion(ctx))
	}

	opts := versioning.DeprecationOptions{
		WarnMessage:     "deprecated, see <this link>",
		DeprecationDate: time.Now().UTC(),
		DeprecationInfo: "a bigger version is available, see <this link> for more information",
	}
	app.Get("/", versioning.Deprecated(writeVesion, opts))

	e := httptest.New(t, app)

	ex := e.GET("/").WithHeader(versioning.AcceptVersionHeaderKey, "1.0").Expect()
	ex.Status(iris.StatusOK).Body().Equal("1.0")
	ex.Header("X-API-Warn").Equal(opts.WarnMessage)
	expectedDateStr := opts.DeprecationDate.Format(app.ConfigurationReadOnly().GetTimeFormat())
	ex.Header("X-API-Deprecation-Date").Equal(expectedDateStr)
}
