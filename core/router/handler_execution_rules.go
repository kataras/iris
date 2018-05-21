package router

import (
	"github.com/kataras/iris/context"
)

// ExecutionRules gives control to the execution of the route handlers outside of the handlers themselves.
// Usage:
// Party#SetExecutionRules(ExecutionRules {
//   Done: ExecutionOptions{Force: true},
// })
//
// See `Party#SetExecutionRules` for more.
type ExecutionRules struct {
	// Begin applies from `Party#Use`/`APIBUilder#UseGlobal` to the first...!last `Party#Handle`'s IF main handlers > 1.
	Begin ExecutionOptions
	// Done applies to the latest `Party#Handle`'s (even if one) and all done handlers.
	Done ExecutionOptions
	// Main applies to the `Party#Handle`'s all handlers, plays nice with the `Done` rule
	// when more than one handler was registered in `Party#Handle` without `ctx.Next()` (for Force: true).
	Main ExecutionOptions
}

func handlersNames(handlers context.Handlers) (names []string) {
	for _, h := range handlers {
		if h == nil {
			continue
		}

		names = append(names, context.HandlerName(h))
	}

	return
}

func applyExecutionRules(rules ExecutionRules, begin, done, main *context.Handlers) {
	if !rules.Begin.Force && !rules.Done.Force && !rules.Main.Force {
		return // do not proceed and spend buld-time here if nothing changed.
	}

	beginOK := rules.Begin.apply(begin)
	mainOK := rules.Main.apply(main)
	doneOK := rules.Done.apply(done)

	if !mainOK {
		mainCp := (*main)[0:]

		lastIdx := len(mainCp) - 1

		if beginOK {
			if len(mainCp) > 1 {
				mainCpFirstButNotLast := make(context.Handlers, lastIdx)
				copy(mainCpFirstButNotLast, mainCp[:lastIdx])

				for i, h := range mainCpFirstButNotLast {
					(*main)[i] = rules.Begin.buildHandler(h)
				}
			}
		}

		if doneOK {
			latestMainHandler := mainCp[lastIdx]
			(*main)[lastIdx] = rules.Done.buildHandler(latestMainHandler)
		}
	}
}

// ExecutionOptions is a set of default behaviors that can be changed in order to customize the execution flow of the routes' handlers with ease.
//
// See `ExecutionRules` and `Party#SetExecutionRules` for more.
type ExecutionOptions struct {
	// Force if true then the handler9s) will execute even if the previous (or/and current, depends on the type of the rule)
	// handler does not calling the `ctx.Next()`,
	// note that the only way remained to stop a next handler is with the `ctx.StopExecution()` if this option is true.
	//
	// If true and `ctx.Next()` exists in the handlers that it shouldn't be, the framework will understand it but use it wisely.
	//
	// Defaults to false.
	Force bool
}

func (e ExecutionOptions) buildHandler(h context.Handler) context.Handler {
	if !e.Force {
		return h
	}

	return func(ctx context.Context) {
		// Proceed will fire the handler and return false here if it doesn't contain a `ctx.Next()`,
		// so we add the `ctx.Next()` wherever is necessary in order to eliminate any dev's misuse.
		if !ctx.Proceed(h) {
			// `ctx.Next()` always checks for `ctx.IsStopped()` and handler(s) positions by-design.
			ctx.Next()
		}
	}
}

func (e ExecutionOptions) apply(handlers *context.Handlers) bool {
	if !e.Force {
		return false
	}

	tmp := *handlers

	for i, h := range tmp {
		if h == nil {
			if len(tmp) == 1 {
				return false
			}
			continue
		}
		(*handlers)[i] = e.buildHandler(h)
	}

	return true
}
