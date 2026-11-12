package mvc_test

import (
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/controller/mvc"
	"github.com/kataras/iris/v14/httptest"

	"github.com/kataras/neffos"
)

type nsWebsocketController struct{}

// Namespace makes the controller implement the (unexported) namespaceController
// interface, so the websocket registrar takes the "has namespace" branch.
func (c *nsWebsocketController) Namespace() string { return "default" }

func (c *nsWebsocketController) OnNamespaceConnected(*neffos.NSConn, neffos.Message) error {
	return nil
}

// TestWebsocketControllerNamespaceRegistration is a regression test for a bug
// where the controller -> namespaceController type assertion was inverted
// (`if nc, ok := c.(namespaceController); !ok`). With the inverted condition,
// building any websocket controller that implemented Namespace() dereferenced a
// nil interface and panicked. Building the application must not panic and the
// websocket route must be registered.
func TestWebsocketControllerNamespaceRegistration(t *testing.T) {
	app := iris.New()
	app.Router("/ws", mvc.Websocket(&nsWebsocketController{}))

	// httptest.New builds the application, which runs the websocket routing
	// closure; the inverted assertion used to panic here.
	e := httptest.New(t, app)

	// A plain GET without the websocket upgrade headers must still reach the
	// websocket handler (not 404), proving the route registered successfully.
	if code := e.GET("/ws/").Expect().Raw().StatusCode; code == 404 {
		t.Fatalf("websocket route was not registered: got status 404")
	}
}
