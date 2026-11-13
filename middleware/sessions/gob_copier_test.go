package sessions_test

import (
	"encoding/gob"
	"errors"
	"reflect"
	"sync"
	"testing"

	"github.com/kataras/iris/v14/middleware/sessions"
)

// Handler copies the data of a session through gob to tell after the request
// whether it changed. The copy is deep, the copier is used again for the next
// request and by other types, and one that failed is not.

type copiedUser struct {
	Name  string
	Roles []string
	Meta  map[string]string
}

func TestCopyThroughGob(t *testing.T) {
	src := sessions.SessionData[copiedUser]{
		Data:          copiedUser{Name: "kataras", Roles: []string{"admin"}, Meta: map[string]string{"theme": "light"}},
		FlashMessages: map[string]any{"notice": "saved"},
		Extra:         "extra",
	}

	for range 3 { // the second and third copy go through a copier that was used before.
		var dst sessions.SessionData[copiedUser]
		if !sessions.CopyThroughGob(src, &dst) {
			t.Fatal("expected the session data to be copied")
		}
		if !reflect.DeepEqual(src, dst) {
			t.Fatalf("expected an equal copy, got %#v", dst)
		}

		// Deep: a change to the copy does not reach the source.
		dst.Data.Roles[0] = "guest"
		dst.Data.Meta["theme"] = "dark"
		dst.FlashMessages["notice"] = "gone"
		if src.Data.Roles[0] != "admin" || src.Data.Meta["theme"] != "light" || src.FlashMessages["notice"] != "saved" {
			t.Fatalf("a change to the copy reached the source: %#v", src)
		}
	}
}

// A value that gob cannot encode is not copied, and the copies after it are
// still right.
func TestCopyThroughGobAfterFailure(t *testing.T) {
	type unencodable struct{ Fn func() }

	for range 3 {
		var bad unencodable
		if sessions.CopyThroughGob(unencodable{Fn: func() {}}, &bad) {
			t.Fatal("expected a value with a func field not to be copied")
		}

		// A destination of another type: the encode succeeds, the decode fails.
		var wrong int
		if sessions.CopyThroughGob(copiedUser{Name: "kataras"}, &wrong) {
			t.Fatal("expected a copy into another type to fail")
		}

		var dst copiedUser
		if !sessions.CopyThroughGob(copiedUser{Name: "kataras", Roles: []string{"admin"}}, &dst) {
			t.Fatal("expected the copy after a failed one to succeed")
		}
		if dst.Name != "kataras" || len(dst.Roles) != 1 || dst.Roles[0] != "admin" {
			t.Fatalf("expected the copy after a failed one to be right, got %#v", dst)
		}
	}
}

// copiedPair holds two values of any type. gob describes the type of such a
// value in the middle of the message, and writes the message to the stream up
// to there.
type copiedPair struct{ First, Second any }

type copiedNote struct{ Text string }

type copiedUnregistered struct{ Text string }

// copiedPicky decodes any value but "bad".
type copiedPicky string

func (p copiedPicky) MarshalBinary() ([]byte, error) { return []byte(p), nil }

func (p *copiedPicky) UnmarshalBinary(b []byte) error {
	if string(b) == "bad" {
		return errors.New("bad value")
	}

	*p = copiedPicky(b)
	return nil
}

type copiedForm struct {
	Field copiedPicky
	Rest  any
}

type copiedRest struct{ Text string }

// A copy that fails halfway leaves a part of its value in the stream of the
// copier: an encode that wrote the start of the message, a decode that did
// not read its end. The copy after it must not read that part.
func TestCopyThroughGobAfterFailureHalfway(t *testing.T) {
	gob.Register(copiedPair{})
	gob.Register(copiedNote{})
	gob.Register(copiedRest{})

	// The encode fails at Second, after the message was written up to First.
	failing := sessions.SessionData[int]{Data: 1, Extra: copiedPair{First: copiedNote{Text: "a"}, Second: copiedUnregistered{Text: "b"}}}
	var pair sessions.SessionData[int]
	if sessions.CopyThroughGob(failing, &pair) {
		t.Fatal("expected a value of an unregistered type not to be copied")
	}

	src := sessions.SessionData[int]{Data: 2, Extra: copiedPair{First: copiedNote{Text: "c"}}}
	pair = sessions.SessionData[int]{}
	if !sessions.CopyThroughGob(src, &pair) || !reflect.DeepEqual(src, pair) {
		t.Fatalf("expected the copy after a failed encode to be right, got %#v", pair)
	}

	// The decode fails at Field, before it read the message that holds Rest.
	var form copiedForm
	if sessions.CopyThroughGob(copiedForm{Field: "bad", Rest: copiedRest{Text: "a"}}, &form) {
		t.Fatal("expected a value that does not decode not to be copied")
	}

	want := copiedForm{Field: "good", Rest: copiedRest{Text: "b"}}
	form = copiedForm{}
	if !sessions.CopyThroughGob(want, &form) || !reflect.DeepEqual(want, form) {
		t.Fatalf("expected the copy after a failed decode to be right, got %#v", form)
	}
}

// Requests copy at the same time, each through a copier of its own.
func TestCopyThroughGobConcurrently(t *testing.T) {
	var wg sync.WaitGroup
	for i := range 64 {
		wg.Go(func() {
			src := copiedUser{Name: "user", Roles: []string{"a", "b"}, Meta: map[string]string{"n": string(rune('a' + i%26))}}
			for range 50 {
				var dst copiedUser
				if !sessions.CopyThroughGob(src, &dst) || !reflect.DeepEqual(src, dst) {
					t.Errorf("expected an equal copy, got %#v", dst)
					return
				}
			}
		})
	}
	wg.Wait()
}
