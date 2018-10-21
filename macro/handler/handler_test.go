package handler

import (
	"testing"

	"github.com/kataras/iris/macro"
)

func TestCanMakeHandler(t *testing.T) {
	tests := []struct {
		src          string
		needsHandler bool
	}{
		{"/static/static", false},
		{"/{myparam}", false},
		{"/{myparam min(1)}", true},
		{"/{myparam else 500}", true},
		{"/{myparam else 404}", false},
		{"/{myparam:string}/static", false},
		{"/{myparam:int}", true},
		{"/static/{myparam:int}/static", true},
		{"/{myparam:path}", false},
		{"/{myparam:path min(1) else 404}", true},
	}

	availableMacros := *macro.Defaults
	for i, tt := range tests {
		tmpl, err := macro.Parse(tt.src, availableMacros)
		if err != nil {
			t.Fatalf("[%d] '%s' failed to be parsed: %v", i, tt.src, err)
		}

		if got := CanMakeHandler(tmpl); got != tt.needsHandler {
			if tt.needsHandler {
				t.Fatalf("[%d] '%s' expected to be able to generate an evaluator handler instead of a nil one", i, tt.src)
			} else {
				t.Fatalf("[%d] '%s' should not need an evaluator handler", i, tt.src)
			}
		}
	}
}
