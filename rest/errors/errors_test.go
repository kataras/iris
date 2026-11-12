package errors_test

import (
	"fmt"
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
	"github.com/kataras/iris/v14/rest/errors"
	"github.com/kataras/iris/v14/rest/errors/validation"
)

type myErr struct {
	Field1           string `json:"field1"`
	Field2           int    `json:"field2"`
	unexported       string
	SkipEmptyField   string `json:"skip_empty_field,omitempty"`
	NoJSONTag        string `json:"-"`
	detail           string
	EmbedStructValue *EmbedStructValue `json:"embed_struct_value"`
}

func (m myErr) Detail() string {
	return m.detail
}

type EmbedStructValue struct {
	EmbedField1 string `json:"embed_field1"`
	EmbedField2 int    `json:"embed_field2"`

	OtherEmbedPtrNil *OtherEmbedPtr `json:"other_embed_ptr_nil,omitempty"`
}

type OtherEmbedPtr struct {
	OtherEmbedPtrField1 string `json:"other_embed_ptr_field1"`
	OtherEmbedPtrField2 int    `json:"other_embed_ptr_field2"`
}

func (e myErr) Error() string {
	return "my custom error"
}

func TestErrorCodeNameRegister(t *testing.T) {
	app := iris.New()

	testError := fmt.Errorf("test error")

	// Register custom error codes for testing.
	TestErrorCode := errors.Register("TEST_ERROR_CODE", iris.StatusBadRequest)

	// Test Log.
	app.Get("/log", func(ctx iris.Context) {
		TestErrorCode.Log(ctx, "test log message")
	})

	// Test LogErr.
	app.Get("/log_err", func(ctx iris.Context) {
		TestErrorCode.LogErr(ctx, testError)
	})

	// Test Message.
	app.Get("/message", func(ctx iris.Context) {
		TestErrorCode.Message(ctx, "test message")
	})

	// Test Detail.
	app.Get("/detail", func(ctx iris.Context) {
		TestErrorCode.Detail(ctx, "test message", "test detail")
	})

	// Test Data.
	app.Get("/data", func(ctx iris.Context) {
		TestErrorCode.Data(ctx, "test message", map[string]string{"key": "value"})
	})

	// Test DataWithDetail.
	app.Get("/data_with_detail", func(ctx iris.Context) {
		TestErrorCode.DataWithDetail(ctx, "test message", "test detail", map[string]string{"key": "value"})
	})

	// Test Validation.
	app.Get("/validation", func(ctx iris.Context) {
		validationErrors := validation.Join(
			validation.Field("age", 17).Func(validation.InRange(18, 200)),
		)
		TestErrorCode.Err(ctx, validationErrors)
	})

	// Test Err.
	app.Get("/err", func(ctx iris.Context) {
		TestErrorCode.Err(ctx, testError)
	})

	// Test ErrPrivate.
	app.Get("/err_private", func(ctx iris.Context) {
		TestErrorCode.ErrPrivate(ctx, testError)
	})

	app.Get("/err_structure_with_json_tags", func(ctx iris.Context) {
		TestErrorCode.Err(ctx, myErr{
			Field1:         "field1_value",
			Field2:         42,
			unexported:     "doesn't matter",
			SkipEmptyField: "",
			NoJSONTag:      "no_json_tag",
			EmbedStructValue: &EmbedStructValue{
				EmbedField1:      "embed_field1_value",
				EmbedField2:      42,
				OtherEmbedPtrNil: nil,
			},
		})
	})

	app.Get("/err_wrapped_structure_with_json_tags", func(ctx iris.Context) {
		TestErrorCode.Err(ctx, fmt.Errorf("parent error: %w", myErr{
			Field1:         "field1_value",
			Field2:         42,
			unexported:     "doesn't matter",
			SkipEmptyField: "",
			NoJSONTag:      "no_json_tag",
			EmbedStructValue: &EmbedStructValue{
				EmbedField1:      "embed_field1_value",
				EmbedField2:      42,
				OtherEmbedPtrNil: nil,
			},
		}))
	})

	app.Get("/err_joined_structure_with_json_tags", func(ctx iris.Context) {
		TestErrorCode.Err(ctx, fmt.Errorf("parent error: %w: %w", ErrDomainLevelError, myErr{
			Field1:         "field1_value",
			Field2:         42,
			unexported:     "doesn't matter",
			SkipEmptyField: "",
			NoJSONTag:      "no_json_tag",
			EmbedStructValue: &EmbedStructValue{
				EmbedField1:      "embed_field1_value",
				EmbedField2:      42,
				OtherEmbedPtrNil: nil,
			},
		}))
	})

	app.Get("/err_joined_simple_wrapped_and_joined_errors", func(ctx iris.Context) {
		nestedErr1 := fmt.Errorf("missing required field firstname")
		nestedErr2 := fmt.Errorf("missing required field lastname")
		nestedErrs := errors.Join(
			nestedErr1,
			nestedErr2,
		)
		TestErrorCode.Err(ctx, fmt.Errorf("user json schema validation failed: %w", nestedErrs))
	})
	/*
	   {
	       ...
	       "message": "user json schema validation failed",
	       "data": [
	           {
	               "message": "missing required field firstname"
	           },
	           {
	               "message": "missing required field lastname"
	           }
	       ]
	   }
	*/
	app.Get("/err_joined_structure_with_json_tags_more_than_one", func(ctx iris.Context) {
		TestErrorCode.Err(ctx, fmt.Errorf("parent error: %w: %w: %w", ErrDomainLevelError, myErr{
			Field1:         "field1_value",
			Field2:         42,
			unexported:     "doesn't matter",
			SkipEmptyField: "",
			NoJSONTag:      "no_json_tag",
			EmbedStructValue: &EmbedStructValue{
				EmbedField1:      "embed_field1_value",
				EmbedField2:      42,
				OtherEmbedPtrNil: nil,
			},
		}, myErr{Field1: "field1_value_2nd_object", detail: "second_has_a_detail_too"}))
	})
	/*
		{
			...
		    "message": "parent error: a domain error: my custom error: my custom error",
		    "data": [
		        {
		            "message": "my custom error",
		            "data": {
		                "field1": "field1_value",
		                "field2": 42,
		                "embed_struct_value": {
		                    "embed_field1": "embed_field1_value",
		                    "embed_field2": 42
		                }
		            }
		        },
		        {
		            "message": "my custom error",
		            "detail": "second_has_a_detail_too",
		            "data": {
		                "field1": "field1_value_2nd_object",
		                "field2": 0,
		                "embed_struct_value": null
		            }
		        }
		    ]
		}
	*/

	e := httptest.New(t, app, httptest.Debug(false))

	// Test Log.
	e.GET("/log").Expect().Status(iris.StatusBadRequest).HasContentType("application/json").JSON().
		Object().Value("http_error_code").Object().ContainsSubset(iris.Map{"canonical_name": "TEST_ERROR_CODE", "status": 400})

	// Test LogErr.
	e.GET("/log_err").Expect().Status(iris.StatusBadRequest).HasContentType("application/json").JSON().
		Object().Value("http_error_code").Object().HasValue("canonical_name", "TEST_ERROR_CODE")

	// Test Message.
	e.GET("/message").Expect().Status(iris.StatusBadRequest).HasContentType("application/json").JSON().
		Object().Value("http_error_code").Object().HasValue("canonical_name", "TEST_ERROR_CODE")

	// Test Detail.
	e.GET("/detail").Expect().Status(iris.StatusBadRequest).HasContentType("application/json").JSON().
		Object().Value("http_error_code").Object().HasValue("canonical_name", "TEST_ERROR_CODE")

	// Test Data.
	e.GET("/data").Expect().Status(iris.StatusBadRequest).HasContentType("application/json").JSON().
		Object().Value("http_error_code").Object().HasValue("canonical_name", "TEST_ERROR_CODE")

	// Test DataWithDetail.
	e.GET("/data_with_detail").Expect().Status(iris.StatusBadRequest).HasContentType("application/json").JSON().
		Object().Value("http_error_code").Object().HasValue("canonical_name", "TEST_ERROR_CODE")

	// Test Validation.
	e.GET("/validation").Expect().Status(iris.StatusBadRequest).HasContentType("application/json").JSON().
		Object().Value("http_error_code").Object().HasValue("canonical_name", "TEST_ERROR_CODE")

	// Test Err.
	e.GET("/err").Expect().Status(iris.StatusBadRequest).HasContentType("application/json").JSON().
		Object().Value("http_error_code").Object().HasValue("canonical_name", "TEST_ERROR_CODE")

	// Test ErrPrivate.
	e.GET("/err_private").Expect().Status(iris.StatusBadRequest).HasContentType("application/json").JSON().
		Object().Value("http_error_code").Object().HasValue("canonical_name", "TEST_ERROR_CODE")

	// Test ErrStructureWithJSONTags.
	errStructureWithJSONTagsTest := e.GET("/err_structure_with_json_tags").Expect().Status(iris.StatusBadRequest).HasContentType("application/json").JSON()
	errStructureWithJSONTagsTest.Object().Value("http_error_code").Object().HasValue("canonical_name", "TEST_ERROR_CODE")
	errStructureWithJSONTagsTest.Object().
		HasValue("message", "my custom error").
		HasValue("data", map[string]any{
			"field1": "field1_value",
			"field2": 42,
			"embed_struct_value": map[string]any{
				"embed_field1": "embed_field1_value",
				"embed_field2": 42,
			},
		})
	// Test ErrWrappedStructureWithJSONTags.
	errWrappedStructureWithJSONTagsTest := e.GET("/err_wrapped_structure_with_json_tags").Expect().Status(iris.StatusBadRequest).HasContentType("application/json").JSON()
	errWrappedStructureWithJSONTagsTest.Object().Value("http_error_code").Object().HasValue("canonical_name", "TEST_ERROR_CODE")
	errWrappedStructureWithJSONTagsTest.Object().
		HasValue("message", "parent error: my custom error").
		HasValue("data", map[string]any{
			"field1": "field1_value",
			"field2": 42,
			"embed_struct_value": map[string]any{
				"embed_field1": "embed_field1_value",
				"embed_field2": 42,
			},
		})
	// Test ErrJoinedStructureWithJSONTags.
	errJoinedStructureWithJSONTagsTest := e.GET("/err_joined_structure_with_json_tags").Expect().Status(iris.StatusBadRequest).HasContentType("application/json").JSON()
	errJoinedStructureWithJSONTagsTest.Object().Value("http_error_code").Object().HasValue("canonical_name", "TEST_ERROR_CODE")
	errJoinedStructureWithJSONTagsTest.Object().
		HasValue("message", "parent error: a domain error: my custom error").
		HasValue("data", map[string]any{ // only one child error with data, no need to give an array of key-values of message:data, as the next test.
			"field1": "field1_value",
			"field2": 42,
			"embed_struct_value": map[string]any{
				"embed_field1": "embed_field1_value",
				"embed_field2": 42,
			},
		})
	// Test ErrJoinedSimpleWrappedAndJoinedErrors.
	errJoinedSimpleWrappedAndJoinedErrorsTest := e.GET("/err_joined_simple_wrapped_and_joined_errors").Expect().Status(iris.StatusBadRequest).HasContentType("application/json").JSON()
	errJoinedSimpleWrappedAndJoinedErrorsTest.Object().Value("http_error_code").Object().HasValue("canonical_name", "TEST_ERROR_CODE")
	errJoinedSimpleWrappedAndJoinedErrorsTest.Object().
		HasValue("message", "user json schema validation failed").
		HasValue("data", []map[string]any{
			{
				"message": "missing required field firstname",
			},
			{
				"message": "missing required field lastname",
			},
		})
		// Test ErrJoinedStructureWithJSONTagsMoreThanOne.
	errJoined2StructureWithJSONTagsTest := e.GET("/err_joined_structure_with_json_tags_more_than_one").Expect().Status(iris.StatusBadRequest).HasContentType("application/json").JSON()
	errJoined2StructureWithJSONTagsTest.Object().Value("http_error_code").Object().HasValue("canonical_name", "TEST_ERROR_CODE")
	errJoined2StructureWithJSONTagsTest.Object().
		HasValue("message", "parent error: a domain error: my custom error: my custom error").
		HasValue("data", []map[string]any{
			{
				"message": "my custom error",
				"data": map[string]any{
					"embed_struct_value": map[string]any{
						"embed_field1": "embed_field1_value",
						"embed_field2": 42,
					},
					"field1": "field1_value",
					"field2": 42,
				},
			},
			{
				"message": "my custom error",
				"detail":  "second_has_a_detail_too",
				"data": map[string]any{
					"embed_struct_value": nil,
					"field1":             "field1_value_2nd_object",
					"field2":             0,
				},
			},
		})
}

func TestErrorCodeNameProblem(t *testing.T) {
	app := iris.New()
	app.Get("/", func(ctx iris.Context) {
		errors.InvalidArgument.Problem(ctx,
			"problem_title",
			"problem_detail",
			"problem_type",
			"problem_instance",
			iris.Map{
				"extra_custom_key":  "extra_custom_value",
				"extra_custom_key2": "extra_custom_value2",
			},
		)
	})

	e := httptest.New(t, app, httptest.Debug(false))
	// Test Problem.
	r := e.GET("/").Expect().Status(iris.StatusBadRequest).JSON(httptest.ContentOpts{MediaType: "application/problem+json"})
	r.Object().Value("status").IsEqual(400)
	r.Object().Value("title").IsEqual("problem_title")
	r.Object().Value("detail").IsEqual("problem_detail")
	r.Object().Value("type").IsEqual("problem_type")
	r.Object().Value("instance").IsEqual("problem_instance")
	r.Object().Value("extra_custom_key").IsEqual("extra_custom_value")
	r.Object().Value("extra_custom_key2").IsEqual("extra_custom_value2")
}
