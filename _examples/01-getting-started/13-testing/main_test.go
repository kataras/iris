// The Bookshelf's test suite. Run with: go test -v
//
// httptest.New(t, app) builds the application (a build failure fails the test
// through t.Fatalf instead of panicking), silences its logs and binds an
// httpexpect client straight to the router. There is no port and no "wait for
// the server", so the suite cannot be flaky for infrastructure reasons. Pass
// httptest.Debug(true) to see every request and response.
package main

import (
	stdContext "context"
	"fmt"
	"testing"

	"uuid"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
)

// testConfig is the configuration the tests choose. Nothing here is a secret,
// and that is the point of passing configuration into newApp instead of
// reading the environment inside it.
func testConfig() Config {
	return Config{
		BaseConfiguration: iris.BaseConfiguration{
			Iris:        iris.DefaultOptions,
			AllowOrigin: "none",
		},
		Env:           "test",
		Developer:     "tests",
		AdminUser:     "admin",
		AdminPassword: "test-password",
	}
}

func TestShelvesCRUD(t *testing.T) {
	e := httptest.New(t, newApp(testConfig(), NewMemStore()))

	// Create. WithJSON marshals the value and sets Content-Type. Raw() steps
	// out of the assertion chain and returns the Go value.
	id := e.POST("/api/shelves").
		WithJSON(ShelfInput{Name: "sci-fi"}).
		Expect().Status(httptest.StatusCreated).
		JSON().Object().Value("id").String().NotEmpty().Raw()

	// Read back.
	e.GET("/api/shelves/"+id).Expect().Status(httptest.StatusOK).
		JSON().Object().HasValue("name", "sci-fi")

	// Validation: ctx.BindJSON runs ShelfInput.Validate, so this is a 400.
	e.POST("/api/shelves").
		WithJSON(ShelfInput{Name: ""}).
		Expect().Status(httptest.StatusBadRequest)

	// The central error map turns ErrNotFound into a 404.
	e.GET("/api/shelves/" + uuid.New().String()).
		Expect().Status(httptest.StatusNotFound)

	// Delete.
	e.DELETE("/api/shelves/" + id).Expect().Status(httptest.StatusNoContent)
}

func TestBooksPaginationAndGuards(t *testing.T) {
	e := httptest.New(t, newApp(testConfig(), NewMemStore()))

	shelfID := e.POST("/api/shelves").
		WithJSON(ShelfInput{Name: "history"}).
		Expect().Status(httptest.StatusCreated).
		JSON().Object().Value("id").String().NotEmpty().Raw()

	for i := 1; i <= 5; i++ {
		e.POST("/api/shelves/" + shelfID + "/books").
			WithJSON(BookInput{Title: fmt.Sprintf("Volume %d", i), Author: "Herodotus", Pages: 100 * i}).
			Expect().Status(httptest.StatusCreated)
	}

	// Page 2 with 2 items per page: items 3 and 4 of 5.
	page := e.GET("/api/shelves/"+shelfID+"/books").
		WithQuery("page", 2).WithQuery("size", 2).
		Expect().Status(httptest.StatusOK).JSON().Object()
	page.HasValue("current_page", 2)
	page.HasValue("total_items", 5)
	page.Value("items").Array().Length().IsEqual(2)

	// A locked shelf rejects writes with 403 but keeps serving reads.
	lockedID := e.POST("/api/shelves").
		WithJSON(ShelfInput{Name: "archive", Locked: true}).
		Expect().Status(httptest.StatusCreated).
		JSON().Object().Value("id").String().NotEmpty().Raw()

	e.POST("/api/shelves/" + lockedID + "/books").
		WithJSON(BookInput{Title: "x", Author: "y", Pages: 1}).
		Expect().Status(httptest.StatusForbidden)
	e.GET("/api/shelves/" + lockedID + "/books").
		Expect().Status(httptest.StatusOK)

	// Unknown shelf: the group guard replies 404 before any handler runs.
	e.GET("/api/shelves/" + uuid.New().String() + "/books").
		Expect().Status(httptest.StatusNotFound)
}

func TestAdminRequiresCredentials(t *testing.T) {
	e := httptest.New(t, newApp(testConfig(), NewMemStore()))

	e.GET("/api/admin/stats").Expect().Status(httptest.StatusUnauthorized)

	e.GET("/api/admin/stats").WithBasicAuth("admin", "test-password").
		Expect().Status(httptest.StatusOK).
		JSON().Object().HasValue("shelves", 0)
}

// failingRepo overrides one method of a working repository; everything else is
// inherited. Interface embedding makes one-method fakes one-liners.
type failingRepo struct {
	Repository
}

func (failingRepo) Shelves(stdContext.Context) ([]Shelf, error) {
	return nil, fmt.Errorf("pg: connection refused on 10.0.0.3:5432")
}

// Internal error details must never reach the client.
func TestInternalErrorsDoNotLeak(t *testing.T) {
	e := httptest.New(t, newApp(testConfig(), failingRepo{NewMemStore()}))

	body := e.GET("/api/shelves").
		Expect().Status(httptest.StatusInternalServerError).
		JSON().Object()
	body.HasValue("message", "server error")
	body.NotContainsValue("10.0.0.3") // the connection string stays server-side.
}
