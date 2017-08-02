package context

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

var (
	topMessages = []string{
		"handler-top-1",
		"handler-top-2",
		"handler-top-3",
		"handler-top-4",
	}

	bottomMessages = []string{
		"handler-bottom-1",
		"handler-bottom-2",
		"handler-bottom-3",
		"handler-bottom-4",
	}
)

type tHandlerTester struct {
	w bytes.Buffer
}

func (tester *tHandlerTester) make_handler(message string) Handler {
	return func(ctx Context) {
		fmt.Fprintln(&tester.w, message)
	}
}

func (tester *tHandlerTester) check(s string) bool {
	return s == tester.w.String()
}

func TestHandlerList_Add_TopFirst(t *testing.T) {
	var tester tHandlerTester

	l := NewHandlerList()

	for i := range topMessages {
		l.AddToTop(tester.make_handler(topMessages[i]))
		l.AddToBottom(tester.make_handler(bottomMessages[i]))
	}

	for _, h := range l.handlers {
		h(nil)
	}

	if !tester.check(fmt.Sprintln(strings.Join(append(topMessages, bottomMessages...), fmt.Sprintln("")))) {
		t.Fail()
	}
}

func TestHandlerList_Add_BottomFirst(t *testing.T) {
	var tester tHandlerTester

	l := NewHandlerList()

	for i := range topMessages {
		l.AddToBottom(tester.make_handler(bottomMessages[i]))
		l.AddToTop(tester.make_handler(topMessages[i]))
	}

	for _, h := range l.handlers {
		h(nil)
	}

	if !tester.check(fmt.Sprintln(strings.Join(append(topMessages, bottomMessages...), fmt.Sprintln("")))) {
		t.Fail()
	}
}

func TestHandlerList_Append_TopFirst(t *testing.T) {
	var tester tHandlerTester
	var topHandlers, bottomHandlers Handlers

	for _, m := range topMessages {
		topHandlers = append(topHandlers, tester.make_handler(m))
	}
	for _, m := range bottomMessages {
		bottomHandlers = append(bottomHandlers, tester.make_handler(m))
	}

	l := NewHandlerList()

	l.AppendToTop(topHandlers[:2])
	l.AppendToBottom(bottomHandlers[:2])
	l.AppendToTop(topHandlers[2:])
	l.AppendToBottom(bottomHandlers[2:])

	for _, h := range l.handlers {
		if h == nil {
			continue
		}
		h(nil)
	}

	if !tester.check(fmt.Sprintln(strings.Join(append(topMessages, bottomMessages...), fmt.Sprintln("")))) {
		t.Fail()
	}
}

func TestHandlerList_Append_BottomFirst(t *testing.T) {
	var tester tHandlerTester
	var topHandlers, bottomHandlers Handlers

	for _, m := range topMessages {
		topHandlers = append(topHandlers, tester.make_handler(m))
	}
	for _, m := range bottomMessages {
		bottomHandlers = append(bottomHandlers, tester.make_handler(m))
	}

	l := NewHandlerList()

	l.AppendToBottom(bottomHandlers[:2])
	l.AppendToTop(topHandlers[:2])
	l.AppendToBottom(bottomHandlers[2:])
	l.AppendToTop(topHandlers[2:])

	for _, h := range l.handlers {
		if h == nil {
			continue
		}
		h(nil)
	}

	if !tester.check(fmt.Sprintln(strings.Join(append(topMessages, bottomMessages...), fmt.Sprintln("")))) {
		t.Fail()
	}
}

func TestHandlerList_Copy(t *testing.T) {
	var tester tHandlerTester

	l := NewHandlerList()

	for i := range topMessages {
		l.AddToBottom(tester.make_handler(bottomMessages[i]))
		l.AddToTop(tester.make_handler(topMessages[i]))
	}

	for _, h := range l.Copy().handlers {
		h(nil)
	}

	if !tester.check(fmt.Sprintln(strings.Join(append(topMessages, bottomMessages...), fmt.Sprintln("")))) {
		t.Fail()
	}
}
