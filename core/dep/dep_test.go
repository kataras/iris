package dep

import (
	stdContext "context"
	"fmt"
	"testing"
)

type (
	Structure struct {
		Name string
	}

	Service interface {
		Print() string
	}
	serviceImpl struct {
		Structure *Structure
		Func      func(v string) string
		// Other     string //  field Other of type string is not binded.
	}
)

func (s *serviceImpl) Print() string {
	return s.Func(s.Structure.Name)
}

func TestContainerRegisterDependencyAndInvoke(t *testing.T) {
	runs := 0

	newService := func(s *Structure, fn func(string) string) *serviceImpl {
		return &serviceImpl{
			Structure: s,
			Func:      fn,
		}
	}
	var deps = []any{
		&Structure{Name: "Gerasimos"},
		func(s *Structure) *Structure { // edge case to change the structure on dependency tree. Only the latest will count from now on.
			return &Structure{
				Name: s.Name + " Maropoulos",
			}
		},
		func() func(string) string {
			return func(v string) string { return v + "!" }
		},
		// &serviceImpl{}, // Service has a nil *Structure field and a Func field, so they should be binded because they are both nil or empty.
		// ^ Same as:
		newService,
		//
		func() string {
			return "Hello, "
		},
		func(s *Structure, prevMsg string) string {
			t.Log(s.Name) // we just print it, it should print just "Gerasimos".
			return prevMsg
		},
		func() int {
			runs++
			return runs // should return 1 always.
		},
		func(helloMsg string, s Service) string { // expecting an interface, has its implementation, so it should be binded.
			return helloMsg + s.Print()
		},
		func(helloMsg string, answer int) string {
			return fmt.Sprintf("%s The answer is: %d", helloMsg, answer)
		},
		// test a request dep.
		func(ctx stdContext.Context) string {
			if ctx == nil {
				t.Fatalf("Expected a context, got nil")
			}
			return "This is a request dependency"
		},
	}

	container := NewContainer()
	for _, dep := range deps {
		err := container.RegisterDependency(dep)
		if err != nil {
			t.Fatalf("Error registering dependency: %v", err)
		}
	}

	fn := func(helloMsg string) string {
		return helloMsg
	}
	result, err := container.InvokeReturningFirst(fn)
	if err != nil {
		t.Fatalf("Error invoking function: %v", err)
	}

	expected := "Hello, Gerasimos Maropoulos! The answer is: 1"
	if result.(string) != expected {
		t.Fatalf("Expected %q, got %q", expected, result)
	}

	// test a request dep with func.
	fn2 := func(fn func(stdContext.Context) string) string {
		return fn(stdContext.Background())
	}

	result, err = container.InvokeReturningFirst(fn2)
	if err != nil {
		t.Fatalf("Error invoking function: %v", err)
	}

	expected = "This is a request dependency"
	if result.(string) != expected {
		t.Fatalf("Expected %q, got %q", expected, result)
	}

	// test a request dep with struct field.
	type funcStruct struct {
		Func func(stdContext.Context) string
	}
	newFuncStruct := func(fn func(stdContext.Context) string) funcStruct {
		return funcStruct{Func: fn}
	}
	result, err = container.InvokeReturningFirst(newFuncStruct)
	if err != nil {
		t.Fatalf("Error invoking function: %v", err)
	}

	st := result.(funcStruct)
	if st.Func == nil {
		t.Fatalf("Expected a non-nil func, got nil")
	}

	if got := st.Func(stdContext.Background()); expected != got {
		t.Fatalf("Expected %q, got %q", expected, got)
	}

	// st and results[0] should be the same.
	if expectedPtr, gotPtr := fmt.Sprintf("%v", st), fmt.Sprintf("%v", result); expectedPtr != gotPtr {
		t.Fatalf("Expected %v, got %v", expectedPtr, gotPtr)
	}
}

func TestContainerRegisterDependency_WithError(t *testing.T) {
	var deps = []any{
		func() (string, error) {
			return "", fmt.Errorf("error")
		},
	}

	container := NewContainer()
	for _, dep := range deps {
		err := container.RegisterDependency(dep)
		if err == nil {
			t.Fatalf("Expected an error, got nil")
		}

		if expected, got := "error", err.Error(); expected != got {
			t.Fatalf("Expected %q, got %q", expected, got)
		}
	}
}

func TestContainerRegisterDependency_WithJustError(t *testing.T) {
	var deps = []any{
		func() float64 {
			return 42
		},
		func(num float64) error {
			return fmt.Errorf("error at: %d", int(num))
		},
	}

	container := NewContainer()
	for i, dep := range deps {
		err := container.RegisterDependency(dep)
		if i > 0 { // the 2nd one shoud fail
			if err == nil {
				t.Fatalf("Expected an error, got nil")
			}

			if expected, got := "error at: 42", err.Error(); expected != got {
				t.Fatalf("Expected %q, got %q", expected, got)
			}
		} else {
			if err != nil {
				t.Fatalf("Expected nil, got %v", err)
			}
		}
	}
}

type Repository[T any] interface {
	Get(string) T
}

type (
	User struct {
		Firstname string
	}

	UserRepository struct{}
)

var _ Repository[User] = (*UserRepository)(nil)

func (r *UserRepository) Get(id string) User {
	firstname := fmt.Sprintf("UserRepository.Get: %s", id)
	return User{Firstname: firstname}
}

func NewUserRepository() *UserRepository {
	return &UserRepository{}
}

type (
	Message struct {
		Body string
	}

	MessageRepository struct{}
)

func (r *MessageRepository) Get(id string) Message {
	body := fmt.Sprintf("MessageRepository.Get: %s", id)
	return Message{Body: body}
}

func NewMessageRepository() *MessageRepository {
	return &MessageRepository{}
}

type AppService struct {
	UserRepo    Repository[User]
	MessageRepo Repository[Message]
}

func NewAppService(messageRepo Repository[Message], userRepo Repository[User] /* make sure input order doesn't matter */) *AppService {
	return &AppService{
		UserRepo:    userRepo,
		MessageRepo: messageRepo,
	}
}

func (s *AppService) Print() string {
	user := s.UserRepo.Get("1")
	message := s.MessageRepo.Get("2")
	return user.Firstname + " - " + message.Body
}

type CustomController struct {
	appService *AppService
	extra      Service
}

type extraService struct{}

func (e *extraService) Print() string {
	return "ExtraService"
}

func NewCustomController(appService *AppService, extra Service) *CustomController {
	return &CustomController{
		appService: appService,
		extra:      extra,
	}
}

func (c *CustomController) Print() string {
	return c.appService.Print() + " - " + c.extra.Print()
}

func TestControllerStyle(t *testing.T) {
	container := NewContainer()
	container.RegisterDependency(NewUserRepository, NewMessageRepository, NewAppService, new(extraService), NewCustomController)

	fn := func(service *CustomController) string {
		return service.Print()
	}
	result, err := container.InvokeReturningFirst(fn)
	if err != nil {
		t.Fatalf("Error invoking function: %v", err)
	}

	if result == nil {
		t.Fatalf("Expected a non-nil result, got nil")
	}

	if _, ok := result.(string); !ok {
		t.Fatalf("Expected a string, got %T", result)
	}

	expected := "UserRepository.Get: 1 - MessageRepository.Get: 2 - ExtraService"
	if got := result.(string); expected != got {
		t.Fatalf("Expected %q, got %q", expected, result)
	}
}
