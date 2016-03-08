package iris

// This file's usage is just to expose the server and it's router functionality
// But the fact is that it has got the only one init func in the project also
import (
	"time"
)

var (
	DefaultServer *Server
)

// The one and only init to the whole package
// set the type of the Context,Renderer and set as default templatesDirectory to the CurrentDirectory of the sources(;)

func init() {
	//Context.go
	//contextType = reflect.TypeOf(Context{})
	//Renderer.go
	//rendererType = reflect.TypeOf(Renderer{})
	//TemplateCache.go
	templatesDirectory = getCurrentDir()

	DefaultServer = New()
}

// IrisOptions contains the available options for the Iris
// Currently these options only used to the Ticker and the Cache.
type IrisOptions struct {
	Cache bool
	// MaxItems max number of total cached routes, 500 = +~20000 bytes = ~0.019073MB
	// Every time the cache timer reach this number it will reset/clean itself
	// Default is 0
	// If <=0 then cache cleans all of items (bag)
	// Auto cache clean is happening after 5 minutes the last request serve, you can change this number by 'ResetDuration' property
	// Note that MaxItems doesn't means that the items never reach this lengh, only on timer tick this number is checked/consider.
	MaxItems int
	// ResetDuration change this time.value to determinate how much duration after last request serving the cache must be reseted/cleaned
	// Default is 5 * time.Minute , Minimum is 30 seconds
	//
	// If MaxItems <= 0 then it clears the whole cache bag at this duration.
	ResetDuration time.Duration
}

func defaultIrisOptions() IrisOptions {
	return IrisOptions{Cache: true, MaxItems: 0, ResetDuration: 5 * time.Minute}
}

// New returns a new iris/server
func New(options ...IrisOptions) *Server {
	_server := new(Server)

	if options != nil && len(options) > 0 {

		if options[0].Cache == false {
			//options has passed and disables the cache
			_server.router = NewRouter()
		} else {
			//options has passed and cache is enabled
			_server.router = NewMemoryRouter(options[0].MaxItems, options[0].ResetDuration)
		}
	} else { // no options has passed to the function, default the options and set MemoryRouter
		opt := defaultIrisOptions()
		_server.router = NewMemoryRouter(opt.MaxItems, opt.ResetDuration) //the default will be the memory router
	}

	return _server
}
