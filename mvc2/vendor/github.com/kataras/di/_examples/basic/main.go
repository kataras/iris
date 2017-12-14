package main

import (
	"fmt"

	"github.com/kataras/di"
)

type User struct {
	Username string
}

type Service interface {
	save(user User) error
}

type testService struct {
	Name string
}

func (s *testService) save(user User) error {
	fmt.Printf("Saving user '%s' from %s Service\n", user.Username, s.Name)
	return nil
}

func myHandler(service Service) {
	service.save(User{
		Username: "func: test username 2",
	})
}

type userManager struct {
	Service Service

	other string
}

func (m *userManager) updateUsername(username string) {
	m.Service.save(User{
		Username: "struct: " + username,
	})
}

func main() {
	// build state, when performance is not critical.
	d := di.New()
	d.Bind(&testService{"test service"})

	structInjector := d.Struct(&userManager{})
	funcInjector := d.Func(myHandler)

	//
	//  at "serve-time", when performance is critical, this DI Binder works very fast.
	//
	// --- for struct's fields ----
	myManager := new(userManager)
	structInjector.Inject(myManager)
	myManager.updateUsername("test username 1")

	// --- for function's input ----
	funcInjector.Call()
}
