package handlers

import (
	"testing"

	"github.com/kataras/iris/context"
)

func TestStackAdd(t *testing.T) {
	l := make([]string, 0)

	stk := &Stack{}
	stk.Add(context.Handlers{
		func(context.Context) {
			l = append(l, "POS1")
		},
	})

	stk.Add(context.Handlers{
		func(context.Context) {
			l = append(l, "POS2")
		},
	})

	if stk.Size() != 2 {
		t.Fatalf("Bad size (%d != 2)", stk.Size())
	}

	for _, h := range stk.List() {
		h(nil)
	}

	if (l[0] != "POS2") || (l[1] != "POS1") {
		t.Fatal("Bad positions: ", l)
	}
}

func TestStackFork(t *testing.T) {
	l := make([]string, 0)

	stk := &Stack{}

	stk.Add(context.Handlers{
		func(context.Context) {
			l = append(l, "POS1")
		},
	})

	stk.Add(context.Handlers{
		func(context.Context) {
			l = append(l, "POS2")
		},
	})

	stk = stk.Fork()

	stk.Add(context.Handlers{
		func(context.Context) {
			l = append(l, "POS3")
		},
	})

	stk.Add(context.Handlers{
		func(context.Context) {
			l = append(l, "POS4")
		},
	})

	if stk.Size() != 4 {
		t.Fatalf("Bad size (%d != 4)", stk.Size())
	}

	for _, h := range stk.List() {
		h(nil)
	}

	if (l[0] != "POS4") || (l[1] != "POS3") || (l[2] != "POS2") || (l[3] != "POS1") {
		t.Fatal("Bad positions: ", l)
	}
}
