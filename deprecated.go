package iris

import (
	"fmt"

	"github.com/kataras/iris/core/router"
	"github.com/kataras/iris/mvc"
)

// Controller method is DEPRECATED, use the "mvc" subpackage instead, i.e
// import "github.com/kataras/iris/mvc" and read its docs among with its new features at:
// https://github.com/kataras/iris/blob/master/HISTORY.md#mo-01-jenuary-2018--v1000
func (app *Application) Controller(relPath string, c interface{}, _ ...interface{}) []*router.Route {
	name := mvc.NameOf(c)

	panic(fmt.Errorf(`"Controller" method is DEPRECATED, use the "mvc" subpackage instead.

        PREVIOUSLY YOU USED TO CODE IT LIKE THIS:
        
            import (
                "github.com/kataras/iris"
                // ...
            )
        
            app.Controller("%s", new(%s), Struct_Values_Binded_To_The_Fields_Or_And_Any_Middleware)
        
        NOW YOU SHOULD CODE IT LIKE THIS:
        
            import (
                "github.com/kataras/iris"
                "github.com/kataras/iris/mvc"
                // ...
            )
        
            // or use it like this:          ).Register(...).Handle(new(%s))
            mvc.Configure(app.Party("%s"), myMVC)
        
            func myMVC(mvcApp *mvc.Application) {
                mvcApp.Register(
                    Struct_Values_Dependencies_Binded_To_The_Fields_Or_And_To_Methods,
                    Or_And_Func_Values_Dependencies_Binded_To_The_Fields_Or_And_To_Methods,
                )
        
                mvcApp.Router.Use(Any_Middleware)
        
                mvcApp.Handle(new(%s))
            }
        
        The new MVC implementation contains a lot more than the above,
        this is the reason you see more lines for a simple controller.
        
        Please read more about the newest, amazing, features by navigating below
        https://github.com/kataras/iris/blob/master/HISTORY.md#mo-01-jenuary-2018--v1000`, // v10.0.0, we skip the number 9.
		relPath, name, name, relPath, name))
}
