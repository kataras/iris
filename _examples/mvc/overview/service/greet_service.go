package service

import (
	"fmt"

	"app/database"
	"app/environment"
)

// GreetService example service.
type GreetService interface {
	Say(input string) (string, error)
}

// NewGreetService returns a service backed with a "db" based on "env".
func NewGreetService(env environment.Env, db database.DB) GreetService {
	service := &greeter{db: db, prefix: "Hello"}

	switch env {
	case environment.PROD:
		return service
	case environment.DEV:
		return &greeterWithLogging{service}
	default:
		panic("unknown environment")
	}
}

type greeter struct {
	prefix string
	db     database.DB
}

func (s *greeter) Say(input string) (string, error) {
	if err := s.db.Exec("simulate a query..."); err != nil {
		return "", err
	}

	result := s.prefix + " " + input
	return result, nil
}

type greeterWithLogging struct {
	*greeter
}

func (s *greeterWithLogging) Say(input string) (string, error) {
	result, err := s.greeter.Say(input)
	fmt.Printf("result: %s\nerror: %v\n", result, err)
	return result, err
}
