package sessions_test

import (
	"testing"
	"time"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/httptest"
	"github.com/kataras/iris/v14/middleware/sessions"

	"uuid"
)

type regenUserSession struct{ Username string }

// memServerDatabase is a minimal in-memory, server-side session database used to
// exercise RegenerateID without a live Redis/PostgreSQL service. It mirrors the
// shipped backends: the session ID is the server-side key and the cookie only
// carries the (encoded) ID.
type memServerDatabase struct {
	opts    sessions.Options
	records map[string]string // "name:id" -> encoded SessionData
}

func newMemServerDatabase(opts sessions.Options) *memServerDatabase {
	return &memServerDatabase{opts: opts.Apply(), records: make(map[string]string)}
}

func (db *memServerDatabase) key(name, id string) string { return name + ":" + id }

func (db *memServerDatabase) Get(ctx *context.Context, name string) (*sessions.Session[regenUserSession], error) {
	return sessions.Load(ctx, db, name)
}

func (db *memServerDatabase) Load(ctx *context.Context, name, value string) (*sessions.Session[regenUserSession], error) {
	session := sessions.NewSession[regenUserSession](db.opts)
	if value == "" {
		session.ID = uuid.New().String()
		session.IsNew = true
		return session, nil
	}

	if err := db.opts.Decode(name, value, &session.ID); err != nil {
		return nil, err
	}
	if encoded, ok := db.records[db.key(name, session.ID)]; ok {
		if err := db.opts.Decode(name, encoded, &session.SessionData); err != nil {
			return nil, err
		}
	} else {
		session.ID = uuid.New().String() // record gone: treat as a fresh session.
	}
	return session, nil
}

func (db *memServerDatabase) Commit(ctx *context.Context, session *sessions.Session[regenUserSession]) error {
	key := db.key(session.Name(), session.ID)
	if session.Options.MaxAge < 0 {
		delete(db.records, key)
		return nil
	}

	encoded, err := db.opts.Encode(session.Name(), session.SessionData)
	if err != nil {
		return err
	}
	db.records[key] = encoded
	return nil
}

// TestSessionRegenerateID verifies that RegenerateID rotates the server-side
// session identifier (fixation defense): the old server record is deleted, the
// data is preserved under a new record with a different id, and a cookie still
// carrying the old id no longer resolves to the authenticated session.
func TestSessionRegenerateID(t *testing.T) {
	opts := sessions.NewOptions(
		[]byte("D*G-KaPdSgUkXp2s5v8y/B?E(H+MbQeThWmYq3t6w9z$C&F)J@NcRfUjXn2r4u7x"),
		[]byte("VkYp3s6v9y$B&E)H@McQfTjWmZq4t7w!"))
	opts.MaxAge = 1 * time.Hour

	db := newMemServerDatabase(opts)

	app := iris.New()

	app.Get("/login", func(ctx iris.Context) {
		session, err := db.Get(ctx, "my_session")
		if err != nil {
			ctx.StopWithText(httptest.StatusInternalServerError, err.Error())
			return
		}
		session.Set(regenUserSession{Username: "kataras"})
		if err := session.RegenerateID(ctx); err != nil {
			ctx.StopWithText(httptest.StatusInternalServerError, err.Error())
			return
		}
		ctx.WriteString("ok")
	})

	app.Get("/whoami", func(ctx iris.Context) {
		session, err := db.Get(ctx, "my_session")
		if err != nil {
			ctx.StopWithText(httptest.StatusInternalServerError, err.Error())
			return
		}
		ctx.WriteString(session.Data.Username)
	})

	e := httptest.New(t, app)

	// Seed a pre-existing (attacker-known) session record under a fixed id and
	// build the cookie value the victim's browser would carry.
	const plantedID = "11111111-1111-1111-1111-111111111111"
	planted, err := db.opts.Encode("my_session", sessions.SessionData[regenUserSession]{})
	if err != nil {
		t.Fatal(err)
	}
	db.records[db.key("my_session", plantedID)] = planted
	plantedCookie, err := db.opts.Encode("my_session", plantedID)
	if err != nil {
		t.Fatal(err)
	}

	// Log in carrying the planted session id.
	e.GET("/login").WithCookie("my_session", plantedCookie).Expect().
		Status(httptest.StatusOK).Body().IsEqual("ok")

	// The planted record must be gone (rotated away).
	if _, ok := db.records[db.key("my_session", plantedID)]; ok {
		t.Fatalf("planted session record was not deleted after RegenerateID")
	}

	// Exactly one (new) record must remain, under a different id.
	if len(db.records) != 1 {
		t.Fatalf("expected exactly one session record after rotation, got %d", len(db.records))
	}
	for key := range db.records {
		if key == db.key("my_session", plantedID) {
			t.Fatalf("rotated session kept the planted id")
		}
	}

	// The planted cookie must no longer resolve to the authenticated session.
	e.GET("/whoami").WithCookie("my_session", plantedCookie).Expect().
		Status(httptest.StatusOK).Body().IsEqual("")
}
