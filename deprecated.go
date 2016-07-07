package iris

import "github.com/kataras/iris/config"

/* Contains some different functions of context.go & iris.go which will be removed on the next revision */

// SecondaryListen same as .AddServer/.Servers.Add(config.Server) instead
// DEPRECATED: use .AddServer instead
// AddServers starts a server which listens to this station
// Note that  the view engine's functions {{ url }} and {{ urlpath }} will return the first's registered server's scheme (http/https)
//
// this is useful mostly when you want to have two or more listening ports ( two or more servers ) for the same station
//
// receives one parameter which is the config.Server for the new server
// returns the new standalone server(  you can close this server by the returning reference)
//
// If you need only one server you can use the blocking-funcs: .Listen/ListenTLS/ListenUNIX/ListenTo
//
// this is a NOT A BLOCKING version, the main .Listen/ListenTLS/ListenUNIX/ListenTo should be always executed LAST, so this function goes before the main .Listen/ListenTLS/ListenUNIX/ListenTo
func SecondaryListen(cfg config.Server) *Server {
	return Default.SecondaryListen(cfg)
}

// SecondaryListen same as .AddServer/.Servers.Add(config.Server) instead
// DEPRECATED: use .AddServer instead
// AddServers starts a server which listens to this station
// Note that  the view engine's functions {{ url }} and {{ urlpath }} will return the first's registered server's scheme (http/https)
//
// this is useful mostly when you want to have two or more listening ports ( two or more servers ) for the same station
//
// receives one parameter which is the config.Server for the new server
// returns the new standalone server(  you can close this server by the returning reference)
//
// If you need only one server you can use the blocking-funcs: .Listen/ListenTLS/ListenUNIX/ListenTo
//
// this is a NOT A BLOCKING version, the main .Listen/ListenTLS/ListenUNIX/ListenTo should be always executed LAST, so this function goes before the main .Listen/ListenTLS/ListenUNIX/ListenTo
func (s *Framework) SecondaryListen(cfg config.Server) *Server {
	return s.Servers.Add(cfg)
}

// NoListen is useful only when you want to test Iris, it doesn't starts the server but it configures and returns it
// DEPRECATED: use ListenVirtual instead
// initializes the whole framework but server doesn't listens to a specific net.Listener
// it is not blocking the app
func NoListen(optionalAddr ...string) *Server {
	return Default.NoListen(optionalAddr...)
}

// NoListen is useful only when you want to test Iris, it doesn't starts the server but it configures and returns it
// DEPRECATED: use ListenVirtual instead
// initializes the whole framework but server doesn't listens to a specific net.Listener
// it is not blocking the app
func (s *Framework) NoListen(optionalAddr ...string) *Server {
	return s.ListenVirtual(optionalAddr...)
}

// CloseWithErr terminates all the registered servers and returns an error if any
// DEPRECATED: use Close instead, and if you want to panic on errors : iris.Must(iris.Close())
// if you want to panic on this error use the iris.Must(iris.Close())
func CloseWithErr() error {
	return Default.Close()
}

// CloseWithErr terminates all the registered servers and returns an error if any
// DEPRECATED: use Close instead, and if you want to panic on errors : iris.Must(iris.Close())
// if you want to panic on this error use the iris.Must(iris.Close())
func (s *Framework) CloseWithErr() error {
	return s.Close()
}

// PostFormMulti returns a slice of string from post request's data
// DEPRECATED: Plase use FormValues instead
func (ctx *Context) PostFormMulti(name string) []string {
	return ctx.FormValues(name)
}

// PostFormValue This will be deprecated
///DEPRECATED: please use FormValueString instead
// PostFormValue returns a single value from post request's data
func (ctx *Context) PostFormValue(name string) string {
	return ctx.FormValueString(name)
}
