package mvc

import (
	"fmt"
	"reflect"

	"github.com/kataras/iris/v14/core/pkg/reflex"
	"github.com/kataras/iris/v14/core/router"
	"github.com/kataras/iris/v14/middleware/websocket"

	"github.com/kataras/neffos"
	"github.com/kataras/neffos/gorilla"
)

// namespaceController is an interface that defines the Namespace method.
// Controllers implementing this interface can specify the namespace for their WebSocket events.
type namespaceController interface {
	// Namespace returns the namespace of the controller.
	Namespace() string
}

// Websocket registers WebSocket controllers and sets up the necessary routes for WebSocket connections and events.
// It accepts a variadic number of controllers, which can be either struct pointers or functions returning struct pointers.
//
// Each controller can implement standard neffos events as methods. The following events can be implemented by the end-developer:
// - OnNamespaceConnect(*neffos.NSConn, neffos.Message) error
// - OnNamespaceConnected(*neffos.NSConn, neffos.Message) error
// - OnNamespaceDisconnect(*neffos.NSConn, neffos.Message) error
// - OnRoomJoin(*neffos.NSConn, neffos.Message) error
// - OnRoomJoined(*neffos.NSConn, neffos.Message) error
// - OnRoomLeave(*neffos.NSConn, neffos.Message) error
// - OnRoomLeft(*neffos.NSConn, neffos.Message) error
//
// Any custom method will be translated as a WebSocket event, e.g.:
// - OnChat(*neffos.NSConn, neffos.Message) error
//
// The function performs the following steps:
// 1. Creates a new neffos.ConnHandler with an empty namespace map.
// 2. Iterates over the provided controllers and resolves them using the dependency injection container.
// 3. Converts each controller to a neffos.StructValue (the reflection form of neffos.Struct) and sets its namespace.
// 4. Binds the controller's fields if it was a struct pointer.
// 5. Joins the controller's event handlers to the neffos.ConnHandler.
// 6. Creates a new neffos.Server with the configured ConnHandler and registers it to the router.
//
// Example usage:
//
//	app.Router("/websocket", mvc.Websocket(newWebsocketController))
//
// This will register the WebSocket controller under the /websocket path.
func Websocket(controllers ...any) router.Controller {
	// router.RegisterControllerResolver(func(c any) (router.Controller, bool) {})
	routing := func(r router.Router) {
		var connHandler neffos.ConnHandler = neffos.Namespaces{}

		defaultNamespace := "default"
		for i, ctrl := range controllers {
			wasStruct := reflex.IsStructPtr(ctrl)

			c, err := r.DI().InvokeReturningFirst(ctrl)
			if err != nil {
				panic(fmt.Errorf("mvc.Websocket: %T: %w", ctrl, err))
			}

			v := reflect.ValueOf(c)
			neffosController := neffos.NewStructValue(v)
			if nc, ok := c.(namespaceController); ok {
				neffosController.SetNamespace(nc.Namespace())

				if i == 0 { // if it's the first controller then set it as the default namespace for the next ones as well.
					defaultNamespace = nc.Namespace()
				}
			} else {
				neffosController.SetNamespace(defaultNamespace)
			}

			if wasStruct { // only if it was struct and not a function which returns a struct, in that case the fields are binded manually through the function/constructor (see InvokeReturningFirst).
				if err := r.DI().BindStructFields(v); err != nil { // v and neffosController.ptr same the same reflect.Value instance.
					panic(fmt.Errorf("mvc.Websocket: %T: %w", c, err))
				}
			}

			connHandler = neffos.JoinConnHandlers(connHandler, neffosController)
		}

		// DBUG: print all registered events.
		// for _, ns := range connHandler.GetNamespaces() {
		// 	for k, v := range ns {
		// 		fmt.Printf("event: %s, callback: %#+v\n", k, v)
		// 	}
		// }

		neffosServer := neffos.New(gorilla.DefaultUpgrader, connHandler)
		neffosHandler := websocket.New(neffosServer, websocket.Options{})

		r.Any("/", neffosHandler)
	}

	return router.Routing(routing)
}
