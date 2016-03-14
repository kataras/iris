package main

// This is just an example may not working at your file system
import (
	"github.com/kataras/iris"
	"github.com/kataras/iris/_examples/plugin_creation/admin_web_interface"
)

func main() {
	webpanelPlugin := admin_web_interface.Newbie()

	iris.Plugin(webpanelPlugin)

	//...
	iris.Listen(":8080")
}
