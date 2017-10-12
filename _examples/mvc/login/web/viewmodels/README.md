# View Models

There should be the view models, the structure that the client will be able to see.

Example:

```go
import (
    "github.com/kataras/iris/_examples/mvc/login/datamodels"

    "github.com/kataras/iris/context"
)

type User struct {
    datamodels.User
}

func (m User) IsValid() bool {
    /* do some checks and return true if it's valid... */
    return m.ID > 0
}
```

Iris is able to convert any custom data Structure into an HTTP Response Dispatcher,
so theoretically, something like the following is permitted if it's really necessary;

```go
// Dispatch completes the `kataras/iris/mvc#Result` interface.
// Sends a `User` as a controlled http response.
// If its ID is zero or less then it returns a 404 not found error
// else it returns its json representation,
// (just like the controller's functions do for custom types by default).
//
// Don't overdo it, the application's logic should not be here.
// It's just one more step of validation before the response,
// simple checks can be added here.
//
// It's just a showcase,
// imagine the potentials this feature gives when designing a bigger application.
//
// This is called where the return value from a controller's method functions
// is type of `User`.
// For example the `controllers/user_controller.go#GetBy`.
func (m User) Dispatch(ctx context.Context) {
    if !m.IsValid() {
        ctx.NotFound()
        return
    }
    ctx.JSON(m, context.JSON{Indent: " "})
}
```

However, we will use the "datamodels" as the only one models package because
User structure doesn't contain any sensitive data, clients are able to see all of its fields
and we don't need any extra functionality or validation inside it.