package sessions_test

import (
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
	"github.com/kataras/iris/v14/middleware/sessions"

	"uuid"
)

// The standard library's uuid.Parse accepts exactly four spellings, the same
// four the previous uuid dependency accepted. Session-id validation and the
// ':uuid' route macro both lean on that leniency, and both are version
// agnostic - the framework now generates v7 session ids while v4 ids issued
// before the switch keep validating. This test pins the accepted and the
// rejected shapes so a future uuid swap cannot silently change either set.
func TestSessionIDParseLeniency(t *testing.T) {
	const canonical = "a6ce7330-d584-4675-af9e-2c2d2c40de07"

	accepted := []struct {
		name string
		id   string
	}{
		{"canonical 36", canonical},
		{"urn prefixed", "urn:uuid:" + canonical},
		{"braced", "{" + canonical + "}"},
		{"bare 32 hex", "a6ce7330d5844675af9e2c2d2c40de07"},
		{"version 1", "c232ab00-9414-11ec-b3c8-9f68deced846"},
		{"version 7", "017f22e2-79b0-7cc3-98c4-dc0c0c07398f"},
	}

	rejected := []struct {
		name string
		id   string
	}{
		{"empty", ""},
		{"too short", "a6ce7330-d584-4675-af9e"},
		{"too long", canonical + "0"},
		{"non hex", "z6ce7330-d584-4675-af9e-2c2d2c40de07"},
		{"misplaced dashes", "a6ce73-30d584-4675-af9e-2c2d2c40de07"},
		{"arbitrary key", "../../etc/passwd"},
	}

	for _, tt := range accepted {
		if !sessions.IsValidSessionID(tt.id) {
			t.Errorf("expected %s (%q) to be a valid session id", tt.name, tt.id)
		}
	}

	for _, tt := range rejected {
		if sessions.IsValidSessionID(tt.id) {
			t.Errorf("expected %s (%q) to be rejected as a session id", tt.name, tt.id)
		}
	}

	// The ':uuid' macro rides the same Parse leniency, routed for real.
	app := iris.New()
	app.Get("/sessions/{id:uuid}", func(ctx iris.Context) {
		ctx.WriteString(ctx.Params().GetString("id"))
	})

	e := httptest.New(t, app)

	for _, tt := range accepted {
		if tt.id == "" {
			continue
		}

		// The macro stores the parsed uuid.UUID, so every spelling reads back canonical.
		e.GET("/sessions/" + tt.id).Expect().Status(iris.StatusOK).Body().IsEqual(uuid.MustParse(tt.id).String())
	}

	for _, tt := range rejected {
		if tt.id == "" || tt.name == "arbitrary key" { // a path with slashes never reaches the macro.
			continue
		}

		e.GET("/sessions/" + tt.id).Expect().Status(iris.StatusNotFound)
	}
}
