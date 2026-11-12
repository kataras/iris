package errors_test

import (
	stdContext "context"
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
	"github.com/kataras/iris/v14/rest/errors"
)

// sortableFilter is a service's own filter: it takes the sort policy by
// embedding iris.ListFilter and nothing else.
type sortableFilter struct {
	iris.ListFilter
	GroupID string `json:"group_id,omitzero" url:"group_id"`
}

// An error code carries the rejection of a sort request, so a list a client
// may read but not order answers that code instead of the default 400.
func TestErrorCodeDenySort(t *testing.T) {
	const message = "sort feature is not available for customers"

	app := iris.New()
	app.Post("/bind", func(ctx iris.Context) {
		_, _, ok := ctx.BindPagination[sortableFilter](errors.PermissionDenied.DenySort(message))
		if !ok {
			return
		}
		ctx.WriteString("ok")
	})
	app.Post("/paginated", func(ctx iris.Context) {
		ctx.Paginated(func(_ stdContext.Context, _ iris.PageOptions, filter sortableFilter) ([]string, int64, error) {
			return []string{filter.GroupID}, 1, nil
		}, errors.PermissionDenied.DenySort(message))
	})
	// The default stays a 400: only a caller that asks for a code gets one.
	app.Post("/default", func(ctx iris.Context) {
		_, _, ok := ctx.BindPagination[sortableFilter](iris.DenySort(message))
		if !ok {
			return
		}
		ctx.WriteString("ok")
	})

	e := httptest.New(t, app)
	post := func(path, body string) *httptest.Request {
		return e.POST(path).WithText(body).WithHeader("Content-Type", "application/json")
	}

	// A sort request is refused with the chosen code and the given message.
	resp := post("/bind", `{"sort":"name"}`).Expect().Status(httptest.StatusForbidden)
	resp.Body().Contains(message)

	post("/paginated", `{"sort":"name"}`).Expect().
		Status(httptest.StatusForbidden).Body().Contains(message)

	// No sort request, no rejection: the policy only answers what was asked.
	post("/bind", `{"group_id":"g1"}`).Expect().
		Status(httptest.StatusOK).Body().IsEqual("ok")
	post("/paginated", `{"group_id":"g1"}`).Expect().
		Status(httptest.StatusOK).Body().Contains("g1")

	// iris.DenySort is unchanged.
	post("/default", `{"sort":"name"}`).Expect().
		Status(httptest.StatusBadRequest).Body().Contains(message)
}
