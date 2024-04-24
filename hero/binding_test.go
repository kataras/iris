package hero

import (
	stdContext "context"
	"fmt"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/kataras/golog"
	"github.com/kataras/iris/v12/context"
	"github.com/kataras/iris/v12/sessions"
)

var (
	stdContextTyp = reflect.TypeOf((*stdContext.Context)(nil)).Elem()
	sessionTyp    = reflect.TypeOf((*sessions.Session)(nil))
	timeTyp       = reflect.TypeOf((*time.Time)(nil)).Elem()
	mapStringsTyp = reflect.TypeOf(map[string][]string{})
)

func contextBinding(index int) *binding {
	return &binding{
		Dependency: BuiltinDependencies[0],
		Input:      &Input{Type: BuiltinDependencies[0].DestType, Index: index},
	}
}

func TestGetBindingsForFunc(t *testing.T) {
	type (
		testResponse struct {
			Name string `json:"name"`
		}

		testRequest struct {
			Email string `json:"email"`
		}

		testRequest2 struct {
			// normally a body can't have two requests but let's test it.
			Age int `json:"age"`
		}
	)

	var testRequestTyp = reflect.TypeOf(testRequest{})

	var deps = []*Dependency{
		NewDependency(func(ctx *context.Context) testRequest { return testRequest{Email: "should be ignored"} }),
		NewDependency(42),
		NewDependency(func(ctx *context.Context) (v testRequest, err error) {
			err = ctx.ReadJSON(&v)
			return
		}),
		NewDependency("if two strings requested this should be the last one"),
		NewDependency("should not be ignored when requested"),

		// Dependencies like these should always be registered last.
		NewDependency(func(ctx *context.Context, input *Input) (newValue reflect.Value, err error) {
			wasPtr := input.Type.Kind() == reflect.Ptr

			newValue = reflect.New(indirectType(input.Type))
			ptr := newValue.Interface()
			err = ctx.ReadJSON(ptr)

			if !wasPtr {
				newValue = newValue.Elem()
			}

			return newValue, err
		}),
	}

	var tests = []struct {
		Func     interface{}
		Expected []*binding
	}{
		{ // 0
			Func: func(ctx *context.Context) {
				ctx.WriteString("t1")
			},
			Expected: []*binding{contextBinding(0)},
		},
		{ // 1
			Func: func(ctx *context.Context) error {
				return fmt.Errorf("err1")
			},
			Expected: []*binding{contextBinding(0)},
		},
		{ // 2
			Func: func(ctx *context.Context) testResponse {
				return testResponse{Name: "name"}
			},
			Expected: []*binding{contextBinding(0)},
		},
		{ // 3
			Func: func(in testRequest) (testResponse, error) {
				return testResponse{Name: "email of " + in.Email}, nil
			},
			Expected: []*binding{{Dependency: deps[2], Input: &Input{Index: 0, Type: testRequestTyp}}},
		},
		{ // 4
			Func: func(in testRequest) (testResponse, error) {
				return testResponse{Name: "not valid "}, fmt.Errorf("invalid")
			},
			Expected: []*binding{{Dependency: deps[2], Input: &Input{Index: 0, Type: testRequestTyp}}},
		},
		{ // 5
			Func: func(ctx *context.Context, in testRequest) testResponse {
				return testResponse{Name: "(with ctx) email of " + in.Email}
			},
			Expected: []*binding{contextBinding(0), {Dependency: deps[2], Input: &Input{Index: 1, Type: testRequestTyp}}},
		},
		{ // 6
			Func: func(in testRequest, ctx *context.Context) testResponse { // reversed.
				return testResponse{Name: "(with ctx) email of " + in.Email}
			},
			Expected: []*binding{{Dependency: deps[2], Input: &Input{Index: 0, Type: testRequestTyp}}, contextBinding(1)},
		},
		{ // 7
			Func: func(in testRequest, ctx *context.Context, in2 string) testResponse { // reversed.
				return testResponse{Name: "(with ctx) email of " + in.Email + "and in2: " + in2}
			},
			Expected: []*binding{
				{
					Dependency: deps[2],
					Input:      &Input{Index: 0, Type: testRequestTyp},
				},
				contextBinding(1),
				{
					Dependency: deps[4],
					Input:      &Input{Index: 2, Type: reflect.TypeOf("")},
				},
			},
		},
		{ // 8
			Func: func(in testRequest, ctx *context.Context, in2, in3 string) testResponse { // reversed.
				return testResponse{Name: "(with ctx) email of " + in.Email + " | in2: " + in2 + " in3: " + in3}
			},
			Expected: []*binding{
				{
					Dependency: deps[2],
					Input:      &Input{Index: 0, Type: testRequestTyp},
				},
				contextBinding(1),
				{
					Dependency: deps[len(deps)-3],
					Input:      &Input{Index: 2, Type: reflect.TypeOf("")},
				},
				{
					Dependency: deps[len(deps)-2],
					Input:      &Input{Index: 3, Type: reflect.TypeOf("")},
				},
			},
		},
		{ // 9
			Func: func(ctx *context.Context, in testRequest, in2 testRequest2) testResponse {
				return testResponse{Name: fmt.Sprintf("(with ctx) email of %s and in2.Age %d", in.Email, in2.Age)}
			},
			Expected: []*binding{
				contextBinding(0),
				{
					Dependency: deps[2],
					Input:      &Input{Index: 1, Type: testRequestTyp},
				},
				{
					Dependency: deps[len(deps)-1],
					Input:      &Input{Index: 2, Type: reflect.TypeOf(testRequest2{})},
				},
			},
		},
		{ // 10
			Func: func() testResponse {
				return testResponse{Name: "empty in, one out"}
			},
			Expected: nil,
		},
		{ // 1
			Func: func(userID string, age int) testResponse {
				return testResponse{Name: "in from path parameters"}
			},
			Expected: []*binding{
				paramBinding(0, 0, reflect.TypeOf("")),
				paramBinding(1, 1, reflect.TypeOf(0)),
			},
		},
		// test std context, session, time, request, response writer and headers  bindings.
		{ // 12
			Func: func(stdContext.Context, *sessions.Session, *golog.Logger, time.Time, *http.Request, http.ResponseWriter, http.Header) testResponse {
				return testResponse{"builtin deps"}
			},
			Expected: []*binding{
				{
					Dependency: NewDependency(BuiltinDependencies[1]),
					Input:      &Input{Index: 0, Type: stdContextTyp},
				},
				{
					Dependency: NewDependency(BuiltinDependencies[2]),
					Input:      &Input{Index: 1, Type: sessionTyp},
				},
				{
					Dependency: NewDependency(BuiltinDependencies[3]),
					Input:      &Input{Index: 2, Type: BuiltinDependencies[3].DestType},
				},
				{
					Dependency: NewDependency(BuiltinDependencies[4]),
					Input:      &Input{Index: 3, Type: timeTyp},
				},
				{
					Dependency: NewDependency(BuiltinDependencies[5]),
					Input:      &Input{Index: 4, Type: BuiltinDependencies[5].DestType},
				},
				{
					Dependency: NewDependency(BuiltinDependencies[6]),
					Input:      &Input{Index: 5, Type: BuiltinDependencies[6].DestType},
				},
				{
					Dependency: NewDependency(BuiltinDependencies[7]),
					Input:      &Input{Index: 6, Type: BuiltinDependencies[7].DestType},
				},
			},
		},
		// test explicitly of http.Header and its underline type map[string][]string which
		// but shouldn't be binded to request headers because of the (.Explicitly()), instead
		// the map should be binded to our last of "deps" which is is a dynamic functions reads from request body's JSON
		// (it's a builtin dependency as well but we declared it to test user dynamic dependencies too).
		{ // 13
			Func: func(http.Header) testResponse {
				return testResponse{"builtin http.Header dep"}
			},
			Expected: []*binding{
				{
					Dependency: NewDependency(BuiltinDependencies[7]),
					Input:      &Input{Index: 0, Type: BuiltinDependencies[7].DestType},
				},
			},
		},
		{ // 14
			Func: func(map[string][]string) testResponse {
				return testResponse{"not dep registered except the dynamic one"}
			},
			Expected: []*binding{
				{
					Dependency: deps[len(deps)-1],
					Input:      &Input{Index: 0, Type: mapStringsTyp},
				},
			},
		},
		{ // 15
			Func: func(http.Header, map[string][]string) testResponse {
				return testResponse{}
			},
			Expected: []*binding{ // only http.Header should be binded, we don't have map[string][]string registered.
				{
					Dependency: NewDependency(BuiltinDependencies[7]),
					Input:      &Input{Index: 0, Type: BuiltinDependencies[7].DestType},
				},
				{
					Dependency: deps[len(deps)-1],
					Input:      &Input{Index: 1, Type: mapStringsTyp},
				},
			},
		},
	}

	c := New()
	for _, dependency := range deps {
		c.Register(dependency)
	}

	for i, tt := range tests {
		bindings := getBindingsForFunc(reflect.ValueOf(tt.Func), c.Dependencies, c.DisablePayloadAutoBinding, 0)

		if expected, got := len(tt.Expected), len(bindings); expected != got {
			t.Fatalf("[%d] expected bindings length to be: %d but got: %d of: %s", i, expected, got, bindings)
		}

		for j, b := range bindings {
			if b == nil {
				t.Fatalf("[%d:%d] binding is nil!", i, j)
			}

			if tt.Expected[j] == nil {
				t.Fatalf("[%d:%d] expected dependency was not found!", i, j)
			}

			// if expected := tt.Expected[j]; !expected.Equal(b) {
			// 	t.Fatalf("[%d:%d] got unexpected binding:\n%s", i, j, spew.Sdump(expected, b))
			// }

			if expected := tt.Expected[j]; !expected.Equal(b) {
				t.Fatalf("[%d:%d] expected binding:\n%s\nbut got:\n%s", i, j, expected, b)
			}
		}
	}
}

type (
	service interface {
		String() string
	}
	serviceImpl struct{}
)

var serviceTyp = reflect.TypeOf((*service)(nil)).Elem()

func (s *serviceImpl) String() string {
	return "service"
}

func TestBindingsForStruct(t *testing.T) {
	type (
		controller struct {
			Name    string
			Service service
		}

		embedded1 struct {
			Age int
		}

		embedded2 struct {
			Now time.Time
		}

		Embedded3 struct {
			Age int
		}

		Embedded4 struct {
			Now time.Time
		}

		controllerEmbeddingExported struct {
			Embedded3
			Embedded4
		}

		controllerEmbeddingUnexported struct {
			embedded1
			embedded2
		}

		controller2 struct {
			Emb1 embedded1
			Emb2 embedded2
		}

		controller3 struct {
			Emb1 embedded1
			emb2 embedded2 // unused
		}
	)

	var deps = []*Dependency{
		NewDependency("name"),
		NewDependency(new(serviceImpl)),
	}

	var depsForAnonymousEmbedded = []*Dependency{
		NewDependency(42),
		NewDependency(time.Now()),
	}

	var depsForFieldsOfStruct = []*Dependency{
		NewDependency(embedded1{Age: 42}),
		NewDependency(embedded2{time.Now()}),
	}

	var depsInterfaces = []*Dependency{
		NewDependency(func(ctx *context.Context) interface{} {
			return "name"
		}),
	}

	var autoBindings = []*binding{
		payloadBinding(0, reflect.TypeOf(embedded1{})),
		payloadBinding(1, reflect.TypeOf(embedded2{})),
	}

	for _, b := range autoBindings {
		b.Input.StructFieldIndex = []int{b.Input.Index}
	}

	var tests = []struct {
		Value      interface{}
		Registered []*Dependency
		Expected   []*binding
	}{
		{ // 0.
			Value:      &controller{},
			Registered: deps,
			Expected: []*binding{
				{
					Dependency: deps[0],
					Input:      &Input{Index: 0, StructFieldIndex: []int{0}, Type: reflect.TypeOf("")},
				},
				{
					Dependency: deps[1],
					Input:      &Input{Index: 1, StructFieldIndex: []int{1}, Type: serviceTyp},
				},
			},
		},
		// 1. test controller with pre-defined variables.
		{
			Value:    &controller{Name: "name_struct", Service: new(serviceImpl)},
			Expected: nil,
		},
		// 2. test controller with pre-defined variables and other deps with the exact order and value
		// (deps from non zero values should be not registerded, if not the Dependency:name_struct will fail for sure).
		{
			Value:      &controller{Name: "name_struct", Service: new(serviceImpl)},
			Registered: deps,
			Expected:   nil,
		},
		// 3. test embedded structs with anonymous and exported.
		{
			Value:      &controllerEmbeddingExported{},
			Registered: depsForAnonymousEmbedded,
			Expected: []*binding{
				{
					Dependency: depsForAnonymousEmbedded[0],
					Input:      &Input{Index: 0, StructFieldIndex: []int{0, 0}, Type: reflect.TypeOf(0)},
				},
				{
					Dependency: depsForAnonymousEmbedded[1],
					Input:      &Input{Index: 1, StructFieldIndex: []int{1, 0}, Type: reflect.TypeOf(time.Time{})},
				},
			},
		},
		// 4. test for anonymous but not exported (should still be 2, unexported structs are binded).
		{
			Value:      &controllerEmbeddingUnexported{},
			Registered: depsForAnonymousEmbedded,
			Expected: []*binding{
				{
					Dependency: depsForAnonymousEmbedded[0],
					Input:      &Input{Index: 0, StructFieldIndex: []int{0, 0}, Type: reflect.TypeOf(0)},
				},
				{
					Dependency: depsForAnonymousEmbedded[1],
					Input:      &Input{Index: 1, StructFieldIndex: []int{1, 0}, Type: reflect.TypeOf(time.Time{})},
				},
			},
		},
		// 5. test for auto-bindings with zero registered.
		{
			Value:      &controller2{},
			Registered: nil,
			Expected:   autoBindings,
		},
		// 6. test for embedded with named fields which should NOT contain any registered deps
		// except the two auto-bindings for structs,
		{
			Value:      &controller2{},
			Registered: depsForAnonymousEmbedded,
			Expected:   autoBindings,
		}, // 7. and only embedded struct's fields are readen, otherwise we expect the struct to be a dependency.
		{
			Value:      &controller2{},
			Registered: depsForFieldsOfStruct,
			Expected: []*binding{
				{
					Dependency: depsForFieldsOfStruct[0],
					Input:      &Input{Index: 0, StructFieldIndex: []int{0}, Type: reflect.TypeOf(embedded1{})},
				},
				{
					Dependency: depsForFieldsOfStruct[1],
					Input:      &Input{Index: 1, StructFieldIndex: []int{1}, Type: reflect.TypeOf(embedded2{})},
				},
			},
		},
		// 8. test one exported and other not exported.
		{
			Value:      &controller3{},
			Registered: []*Dependency{depsForFieldsOfStruct[0]},
			Expected: []*binding{
				{
					Dependency: depsForFieldsOfStruct[0],
					Input:      &Input{Index: 0, StructFieldIndex: []int{0}, Type: reflect.TypeOf(embedded1{})},
				},
			},
		},
		// 9. test same as the above but by registering all dependencies.
		{
			Value:      &controller3{},
			Registered: depsForFieldsOfStruct,
			Expected: []*binding{
				{
					Dependency: depsForFieldsOfStruct[0],
					Input:      &Input{Index: 0, StructFieldIndex: []int{0}, Type: reflect.TypeOf(embedded1{})},
				},
			},
		},
		// 10. test bind an interface{}.
		{
			Value:      &controller{},
			Registered: depsInterfaces,
			Expected: []*binding{
				{
					Dependency: depsInterfaces[0],
					Input:      &Input{Index: 0, StructFieldIndex: []int{0}, Type: reflect.TypeOf("")},
				},
			},
		},
	}

	for i, tt := range tests {
		bindings := getBindingsForStruct(reflect.ValueOf(tt.Value), tt.Registered, false, false, false, DefaultDependencyMatcher, 0, nil)

		if expected, got := len(tt.Expected), len(bindings); expected != got {
			t.Logf("[%d] expected bindings length to be: %d but got: %d:\n", i, expected, got)
			for _, b := range bindings {
				t.Logf("\t%s\n", b)
			}
			t.FailNow()
		}

		for j, b := range bindings {
			if tt.Expected[j] == nil {
				t.Fatalf("[%d:%d] expected dependency was not found!", i, j)
			}

			if expected := tt.Expected[j]; !expected.Equal(b) {
				t.Fatalf("[%d:%d] expected binding:\n%s\nbut got:\n%s", i, j, expected, b)
			}
		}
	}

}

func TestBindingsForStructMarkExportedFieldsAsRequred(t *testing.T) {
	type (
		Embedded struct {
			Val string
		}

		controller struct {
			MyService service
			Embedded  *Embedded
		}
	)

	dependencies := []*Dependency{
		NewDependency(&Embedded{"test"}),
		NewDependency(&serviceImpl{}),
	}

	// should panic if fail.
	_ = getBindingsForStruct(reflect.ValueOf(new(controller)), dependencies, true, true, false, DefaultDependencyMatcher, 0, nil)
}
