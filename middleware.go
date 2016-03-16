package iris

// Middleware is just a slice of Handler []func(c *Context)
type Middleware []Handler

// getLast returns the last registed Handler, for a route this is the route's main handler
func (m Middleware) getLast() Handler {
	if m == nil || len(m) == 0 {
		return nil
	}

	return m[len(m)-1]
}

type IMiddlewareSupporter interface {
	Use(handlers ...Handler)
	UseFunc(handlersFn ...HandlerFunc)
}

type MiddlewareSupporter struct {
	middleware Middleware
}

// joinMiddleware uses to create a copy of all middleware and return them in order to use inside the node
func (m MiddlewareSupporter) joinMiddleware(middleware Middleware) Middleware {
	nowLen := len(m.middleware)
	totalLen := nowLen + len(middleware)
	// create a new slice of middleware in order to store all handlers, the already handlers(middleware) and the new
	newMiddleware := make(Middleware, totalLen)
	//copy the already middleware to the just created
	copy(newMiddleware, m.middleware)
	//start from there we finish, and store the new middleware too
	copy(newMiddleware[nowLen:], middleware)
	return newMiddleware
}

// Use appends handler(s) to the route or to the router if it's called from router
func (m MiddlewareSupporter) Use(handlers ...Handler) {
	m.middleware = append(m.middleware, handlers...)
}

// UseFunc is the same as Use but it receives HandlerFunc instead of iris.Handler as parameter(s)
// form of acceptable: func(c *iris.Context){//first middleware}, func(c *iris.Context){//second middleware}
func (m MiddlewareSupporter) UseFunc(handlersFn ...HandlerFunc) {
	for _, h := range handlersFn {
		m.Use(Handler(h))
	}
}
