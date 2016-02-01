package middleware

import (
	"net/http"
)

type Handler func(http.Handler) http.Handler

type Middleware struct {
	Method   string
	Handlers []Handler
	Pattern string
}