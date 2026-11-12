package errors_test

import (
	"fmt"
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
	"github.com/kataras/iris/v14/rest/errors"
)

func TestHandler(t *testing.T) {
	app := iris.New(iris.Options{
		Router: iris.RouterOptions{
			FireMethodNotAllowed: true,
		},
	})
	app.Use(errors.Handler)
	// Should not call EnableStatusCodeResponse here, because we want to test the default error handlers and that will break it.
	// Maybe a future fix, will test the EnableStatusCodeResponse too. Or by checking the registered middlewares and if the errors.Handler
	// is registered then we don't register the underline EnableStatusCodeResponse's middleware.
	// app.EnableStatusCodeResponse()

	var (
		indexError   = fmt.Errorf("index error")
		blockedError = fmt.Errorf("blocked by windows parental controls")

		expectedJSONError = errors.Error{
			ErrorCode: errors.ErrorCode{
				CanonicalName: errors.UnsupportedMediaType,
				Status:        415,
			},
			Message: indexError.Error(),
		}

		expectedNotFoundNoRouteJSONError = errors.Error{
			ErrorCode: errors.ErrorCode{
				CanonicalName: errors.NotFound,
				Status:        404,
			},
			Message: "Not Found",
			Detail:  "The requested page could not be found.",
		}

		expectedMethodNotAllowedJSONError = errors.Error{
			ErrorCode: errors.ErrorCode{
				CanonicalName: errors.MethodNotAllowed,
				Status:        405,
			},
			Message: "Method Not Allowed",
			Detail:  "Allowed methods: GET, POST",
		}

		expectedBlockedByWindowsParentalControlsJSONError = errors.Error{
			ErrorCode: errors.ErrorCode{
				CanonicalName: "BLOCKED_BY_WINDOWS_PARENTAL_CONTROLS",
				Status:        450,
			},
			Message: blockedError.Error(),
		}

		// This is when an error is missing but a fail status code exists. Acts like EnableStatusCodeResponse
		// managed by the handler wrapper itself (err is not set, is created by the status code's text, like the canonical name).
		expectedPlainInsufficientStorageError = errors.Error{
			ErrorCode: errors.ErrorCode{
				CanonicalName: "INSUFFICIENT_STORAGE",
				Status:        507,
			},
			Message: "Insufficient Storage",
		}
	)

	app.Handle("GET POST", "/", func(ctx iris.Context) {
		ctx.StopWithPlainError(iris.StatusUnsupportedMediaType, indexError)
	})

	// This is not a registered error code, but it should be created at-serve time and see expected JSON result.
	app.Get("/450", func(ctx iris.Context) {
		ctx.StopWithPlainError(iris.StatusBlockedByWindowsParentalControls, blockedError)
	})

	app.Get("/507", func(ctx iris.Context) {
		ctx.StopWithStatus(iris.StatusInsufficientStorage)
	})

	e := httptest.New(t, app)
	e.GET("/").Expect().Status(iris.StatusUnsupportedMediaType).
		JSON().Object().ContainsSubset(expectedJSONError)

	e.GET("/404").Expect().Status(iris.StatusNotFound).
		JSON().Object().ContainsSubset(expectedNotFoundNoRouteJSONError)

	e.PUT("/").Expect().Status(iris.StatusMethodNotAllowed).
		JSON().Object().ContainsSubset(expectedMethodNotAllowedJSONError)

	e.GET("/450").Expect().Status(iris.StatusBlockedByWindowsParentalControls).
		JSON().Object().ContainsSubset(expectedBlockedByWindowsParentalControlsJSONError)

	e.GET("/507").Expect().Status(iris.StatusInsufficientStorage).
		JSON().Object().ContainsSubset(expectedPlainInsufficientStorageError)
}
