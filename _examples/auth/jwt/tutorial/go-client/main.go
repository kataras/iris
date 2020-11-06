package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"

	"myapp/api"
	"myapp/domain/model"
)

const base = "http://localhost:8080"

func main() {
	accessToken, err := authenticate("admin", "admin")
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Access Token:\n%q", accessToken)

	todo, err := createTodo(accessToken, "test todo title", "test todo body contents")
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Todo Created:\n%#+v", todo)
}

func authenticate(username, password string) ([]byte, error) {
	endpoint := base + "/signin"

	data := make(url.Values)
	data.Set("username", username)
	data.Set("password", password)

	resp, err := Form(http.MethodPost, endpoint, data)
	if err != nil {
		return nil, err
	}

	accessToken, err := RawResponse(resp)
	return accessToken, err
}

func createTodo(accessToken []byte, title, body string) (model.Todo, error) {
	var todo model.Todo

	endpoint := base + "/todos"

	req := api.TodoRequest{
		Title: title,
		Body:  body,
	}

	resp, err := JSON(http.MethodPost, endpoint, req, WithAccessToken(accessToken))
	if err != nil {
		return todo, err
	}

	if resp.StatusCode != http.StatusCreated {
		rawData, _ := RawResponse(resp)
		return todo, fmt.Errorf("failed to create a todo: %s", string(rawData))
	}

	err = BindResponse(resp, &todo)
	return todo, err
}
