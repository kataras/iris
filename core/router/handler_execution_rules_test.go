package router_test

import (
	"testing"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/context"
	"github.com/kataras/iris/v12/core/router"
	"github.com/kataras/iris/v12/httptest"
)

var (
	finalExecutionRulesResponse = "1234"

	testExecutionResponse = func(t *testing.T, app *iris.Application, path string) {
		e := httptest.New(t, app)
		e.GET(path).Expect().Status(httptest.StatusOK).Body().IsEqual(finalExecutionRulesResponse)
	}
)

func writeStringHandler(text string, withNext bool) context.Handler {
	return func(ctx *context.Context) {
		ctx.WriteString(text)
		if withNext {
			ctx.Next()
		}
	}
}

func TestRouterExecutionRulesForceMain(t *testing.T) {
	app := iris.New()
	begin := app.Party("/")
	begin.SetExecutionRules(router.ExecutionRules{Main: router.ExecutionOptions{Force: true}})

	// no need of `ctx.Next()` all main handlers should be executed with the Main.Force:True rule.
	begin.Get("/", writeStringHandler("12", false), writeStringHandler("3", false), writeStringHandler("4", false))

	testExecutionResponse(t, app, "/")
}

func TestRouterExecutionRulesForceBegin(t *testing.T) {
	app := iris.New()
	begin := app.Party("/begin_force")
	begin.SetExecutionRules(router.ExecutionRules{Begin: router.ExecutionOptions{Force: true}})

	// should execute, begin rule is to force execute them without `ctx.Next()`.
	begin.Use(writeStringHandler("1", false))
	begin.Use(writeStringHandler("2", false))
	// begin starts with begin and ends to the main handlers but not last, so this done should not be executed.
	begin.Done(writeStringHandler("5", false))
	begin.Get("/", writeStringHandler("3", false), writeStringHandler("4", false))

	testExecutionResponse(t, app, "/begin_force")
}

func TestRouterExecutionRulesForceDone(t *testing.T) {
	app := iris.New()
	done := app.Party("/done_force")
	done.SetExecutionRules(router.ExecutionRules{Done: router.ExecutionOptions{Force: true}})

	// these done should be executed without `ctx.Next()`
	done.Done(writeStringHandler("3", false), writeStringHandler("4", false))
	// first with `ctx.Next()`, because Done.Force:True rule will alter the latest of the main handler(s) only.
	done.Get("/", writeStringHandler("1", true), writeStringHandler("2", false))

	// rules should be kept in children.
	doneChild := done.Party("/child")
	// even if only one, it's the latest, Done.Force:True rule should modify it.
	doneChild.Get("/", writeStringHandler("12", false))

	testExecutionResponse(t, app, "/done_force")
	testExecutionResponse(t, app, "/done_force/child")
}

func TestRouterExecutionRulesShouldNotModifyTheCallersHandlerAndChildrenCanResetExecutionRules(t *testing.T) {
	app := iris.New()
	app.SetExecutionRules(router.ExecutionRules{Done: router.ExecutionOptions{Force: true}})
	h := writeStringHandler("4", false)

	app.Done(h)
	app.Get("/", writeStringHandler("123", false))

	// remember: the handler stored in var didn't had a `ctx.Next()`, modified its clone above with adding a `ctx.Next()`
	// note the "clone" word, the original handler shouldn't be changed.
	app.Party("/c").SetExecutionRules(router.ExecutionRules{}).Get("/", h, writeStringHandler("err caller modified!", false))

	testExecutionResponse(t, app, "/")

	e := httptest.New(t, app)
	e.GET("/c").Expect().Status(httptest.StatusOK).Body().IsEqual("4") // the "should not" should not be written.
}
