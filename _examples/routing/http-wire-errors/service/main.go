package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/x/errors"
	"github.com/kataras/iris/v12/x/errors/validation"
	"github.com/kataras/iris/v12/x/pagination"
)

func main() {
	app := iris.New()

	// Create a new service and pass it to the handlers.
	service := new(myService)

	app.Post("/", errors.Intercept(afterServiceCallButBeforeDataSent), createHandler(service)) // OR: errors.CreateHandler(service.Create)
	app.Get("/", listAllHandler(service))                                                      // OR errors.Handler(service.ListAll, errors.Value(ListRequest{}))
	app.Post("/page", listHandler(service))                                                    // OR: errors.ListHandler(service.ListPaginated)
	app.Delete("/{id:string}", deleteHandler(service))                                         // OR: errors.NoContentOrNotModifiedHandler(service.DeleteWithFeedback, errors.PathParam[string]("id"))

	app.Listen(":8080")
}

func createHandler(service *myService) iris.Handler {
	return func(ctx iris.Context) {
		// What it does?
		// 1. Reads the request body and binds it to the CreateRequest struct.
		// 2. Calls the service.Create function with the given request body.
		// 3. If the service.Create returns an error, it sends an appropriate error response to the client.
		// 4. If the service.Create returns a response, it sets the status code to 201 (Created) and sends the response as a JSON payload to the client.
		//
		// Useful for create operations.
		errors.Create(ctx, service.Create)
	}
}

func listAllHandler(service *myService) iris.Handler {
	return func(ctx iris.Context) {
		// What it does?
		// 1. If the 3rd variadic (optional) parameter is empty (not our case here), it reads the request body and binds it to the ListRequest struct,
		// otherwise (our case) it calls the service.ListAll function directly with the given input parameter (empty ListRequest struct value in our case).
		// 2. Calls the service.ListAll function with the ListRequest value.
		// 3. If the service.ListAll returns an error, it sends an appropriate error response to the client.
		// 4. If the service.ListAll returns a response, it sets the status code to 200 (OK) and sends the response as a JSON payload to the client.
		//
		// Useful for get single, fetch multiple and search operations.
		errors.OK(ctx, service.ListAll, ListRequest{})
	}
}

func listHandler(service *myService) iris.Handler {
	return func(ctx iris.Context) {
		errors.List(ctx, service.ListPaginated)
	}
}

func deleteHandler(service *myService) iris.Handler {
	return func(ctx iris.Context) {
		id := ctx.Params().Get("id")
		// What it does?
		// 1. Calls the service.DeleteWithFeedback function with the given input parameter.
		// 2. If the service.DeleteWithFeedback returns an error, it sends an appropriate error response to the client.
		// 3.If the service.DeleteWithFeedback doesn't return an error then it sets the status code to 204 (No Content) and
		// sends the response as a JSON payload to the client.
		// errors.NoContent(ctx, service.Delete, id)
		// OR:
		// 1. Calls the service.DeleteWithFeedback function with the given input parameter.
		// 2. If the service.DeleteWithFeedback returns an error, it sends an appropriate error response to the client.
		// 3. If the service.DeleteWithFeedback returns true, it sets the status code to 204 (No Content).
		// 4. If the service.DeleteWithFeedback returns false, it sets the status code to 304 (Not Modified).
		//
		// Useful for update and delete operations.
		errors.NoContentOrNotModified(ctx, service.DeleteWithFeedback, id)
	}
}

type (
	myService struct{}

	CreateRequest struct {
		Fullname string   `json:"fullname"`
		Age      int      `json:"age"`
		Hobbies  []string `json:"hobbies"`
	}

	CreateResponse struct {
		ID        string   `json:"id"`
		Firstname string   `json:"firstname"`
		Lastname  string   `json:"lastname"`
		Age       int      `json:"age"`
		Hobbies   []string `json:"hobbies"`
	}
)

// HandleRequest implements the errors.RequestHandler interface.
// It validates the request body and returns an error if the request body is invalid.
// You can also alter the "r" CreateRequest before calling the service method,
// e.g. give a default value to a field if it's empty or set an ID based on a path parameter.
// OR
// Custom function per route:
//
//	r.Post("/", errors.Validation(validateCreateRequest), createHandler(service))
//	[more code here...]
//
//	func validateCreateRequest(ctx iris.Context, r *CreateRequest) error {
//		return validation.Join(
//			validation.String("fullname", r.Fullname).NotEmpty().Fullname().Length(3, 50),
//			validation.Number("age", r.Age).InRange(18, 130),
//			validation.Slice("hobbies", r.Hobbies).Length(1, 10),
//		)
//	}
func (r *CreateRequest) HandleRequest(ctx iris.Context) error {
	// To pass custom validation functions:
	// return validation.Join(
	// 	validation.String("fullname", r.Fullname).Func(customStringFuncHere),
	//   OR
	// 	validation.Field("any_field", r.AnyFieldValue).Func(customAnyFuncHere))
	return validation.Join(
		validation.String("fullname", r.Fullname).Fullname().Length(3, 50),
		validation.Number("age", r.Age).InRange(18, 130),
		validation.Slice("hobbies", r.Hobbies).Length(1, 10),
	)

	/* Example Output:
	{
	    "http_error_code": {
	        "canonical_name": "INVALID_ARGUMENT",
	        "status": 400
	    },
	    "message": "validation failure",
	    "details": "fields were invalid",
	    "validation": [
	        {
	            "field": "fullname",
	            "value": "",
	            "reason": "must not be empty, must contain first and last name, must be between 3 and 50 characters"
	        },
	        {
	            "field": "age",
	            "value": 0,
	            "reason": "must be in range of [18, 130]"
	        },
	        {
	            "field": "hobbies",
	            "value": null,
	            "reason": "must be between 1 and 10 elements"
	        }
	    ]
	}
	*/
}

/*
// HandleResponse implements the errors.ResponseHandler interface.
func (r *CreateRequest) HandleResponse(ctx iris.Context, resp *CreateResponse) error {
	fmt.Printf("request got: %+v\nresponse sent: %#+v\n", r, resp)

	return nil // fmt.Errorf("let's fire an internal server error just for the shake of the example") // return nil to continue.
}
*/

func afterServiceCallButBeforeDataSent(ctx iris.Context, req CreateRequest, resp *CreateResponse) error {
	fmt.Printf("intercept: request got: %+v\nresponse sent: %#+v\n", req, resp)
	return nil
}

func (s *myService) Create(ctx context.Context, in CreateRequest) (CreateResponse, error) {
	arr := strings.Split(in.Fullname, " ")
	firstname, lastname := arr[0], arr[1]
	id := "test_id"

	resp := CreateResponse{
		ID:        id,
		Firstname: firstname,
		Lastname:  lastname,
		Age:       in.Age,
		Hobbies:   in.Hobbies,
	}
	return resp, nil // , errors.New("create: test error")
}

type ListRequest struct {
}

func (s *myService) ListAll(ctx context.Context, in ListRequest) ([]CreateResponse, error) {
	resp := []CreateResponse{
		{
			ID:        "test-id-1",
			Firstname: "test first name 1",
			Lastname:  "test last name 1",
		},
		{
			ID:        "test-id-2",
			Firstname: "test first name 2",
			Lastname:  "test last name 2",
		},
		{
			ID:        "test-id-3",
			Firstname: "test first name 3",
			Lastname:  "test last name 3",
		},
	}

	return resp, nil //, errors.New("list all: test error")
}

type ListFilter struct {
	Firstname string `json:"firstname"`
}

func (s *myService) ListPaginated(ctx context.Context, opts pagination.ListOptions, filter ListFilter) ([]CreateResponse, int /* any number type */, error) {
	all, err := s.ListAll(ctx, ListRequest{})
	if err != nil {
		return nil, 0, err
	}

	filteredResp := make([]CreateResponse, 0)
	for _, v := range all {
		if strings.Contains(v.Firstname, filter.Firstname) {
			filteredResp = append(filteredResp, v)
		}

		if len(filteredResp) == opts.GetLimit() {
			break
		}
	}

	return filteredResp, len(all), nil // errors.New("list paginated: test error")
}

func (s *myService) GetByID(ctx context.Context, id string) (CreateResponse, error) {
	return CreateResponse{Firstname: "Gerasimos"}, nil // errors.New("get by id: test error")
}

func (s *myService) Delete(ctx context.Context, id string) error {
	return nil // errors.New("delete: test error")
}

func (s *myService) Update(ctx context.Context, req CreateRequest) (bool, error) {
	return true, nil // false, errors.New("update: test error")
}

func (s *myService) DeleteWithFeedback(ctx context.Context, id string) (bool, error) {
	return true, nil // false, errors.New("delete: test error")
}
