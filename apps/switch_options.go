package apps

import "net/http"

type (
	// SwitchOptions holds configuration
	// for the switcher application.
	SwitchOptions struct {
		// RequestModifiers holds functions to run
		// if and only if at least one Filter passed.
		// They are used to modify the request object
		// of the matched Application, e.g. modify the host.
		//
		// See `SetHost` option too.
		RequestModifiers []func(*http.Request)
		// Note(@kataras): I though a lot of API designs for that one and the current is the safest to use.
		// I skipped the idea of returning a wrapped Application to have functions like app.UseFilter
		// or the idea of accepting a chain of Iris Handlers here because the Context belongs
		// to the switcher application and a new one is acquired on the matched Application level,
		// so communication between them is not possible although
		// we can make it possible but lets not complicate the code here, unless otherwise requested.
	}

	// SwitchOption should be implemented by all options
	// passed to the `Switch` package-level last variadic input argument.
	SwitchOption interface {
		Apply(*SwitchOptions)
	}

	// SwitchOptionFunc provides a functional way to pass options
	// to the `Switch` package-level function's last variadic input argument.
	SwitchOptionFunc func(*SwitchOptions)
)

// Apply completes the `SwitchOption` interface.
func (f SwitchOptionFunc) Apply(opts *SwitchOptions) {
	f(opts)
}

// DefaultSwitchOptions returns a fresh SwitchOptions
// struct value with its fields set to their defaults.
func DefaultSwitchOptions() SwitchOptions {
	return SwitchOptions{
		RequestModifiers: nil,
	}
}

// Apply completes the `SwitchOption` interface.
// It does copies values from "o" to "opts" when necessary.
func (o SwitchOptions) Apply(opts *SwitchOptions) {
	if v := o.RequestModifiers; len(v) > 0 {
		opts.RequestModifiers = v // override, not append.
	}
}

// SetHost is a SwitchOption.
// It force sets a Host field for the matched Application's request object.
// Extremely useful when used with Hosts SwitchProvider.
// Usecase: www. to root domain without redirection (SEO reasons)
// and keep the same internal request Host for both of them so
// the root app's handlers will always work with a single host no matter
// what the real request Host was.
func SetHost(hostField string) SwitchOptionFunc {
	if hostField == "" {
		return nil
	}

	setHost := func(r *http.Request) {
		r.Host = hostField
		r.URL.Host = hostField // note: the URL.String builds the uri based on that.
	}

	return func(opts *SwitchOptions) {
		opts.RequestModifiers = append(opts.RequestModifiers, setHost)
	}
}
