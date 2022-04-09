package modrevision

import (
	"fmt"
	"strings"
	"time"

	"github.com/kataras/iris/v12/context"
)

func init() {
	context.SetHandlerName("iris/middleware/modrevision.*", "iris.modrevision")
}

// Options holds the necessary values to render the server name, environment and build information.
// See the `New` package-level function.
type Options struct {
	// The ServerName, e.g. Iris Server.
	ServerName string
	// The Environment, e.g. development.
	Env string
	// The Developer, e.g. kataras.
	Developer string
	// True to display the build time as unix (seconds).
	UnixTime bool
	// A non nil time location value to customize the display of the build time.
	TimeLocation *time.Location
}

type modRevision struct {
	options Options

	buildTime     string
	buildRevision string

	contents []byte
}

// New returns an Iris Handler which renders
// the server name (env), build information (if available)
// and an OK message. The handler displays simple debug information such as build commit id and time.
// It does NOT render information about the Go language itself or any operating system confgiuration
// for security reasons.
//
// Example Code:
//  app.Get("/health", modrevision.New(modrevision.Options{
//   ServerName:   "Iris Server",
//   Env:          "development",
//   Developer:    "kataras",
//   TimeLocation: time.FixedZone("Greece/Athens", 10800),
//  }))
func New(opts Options) context.Handler {
	bTime, bRevision := context.BuildTime, context.BuildRevision
	if opts.UnixTime {
		if t, err := time.Parse(time.RFC3339, bTime); err == nil {
			bTime = fmt.Sprintf("%d", t.Unix())
		}
	} else if opts.TimeLocation != nil {
		if t, err := time.Parse(time.RFC3339, bTime); err == nil {
			bTime = t.In(opts.TimeLocation).String()
		}
	}

	m := &modRevision{
		options: opts,

		buildTime:     bTime,
		buildRevision: bRevision,
	}

	contents := []byte(m.String())
	if len(contents) > 0 {
		contents = append(contents, []byte("\n\nOK")...)
	} else {
		contents = []byte("OK")
	}

	return func(ctx *context.Context) {
		ctx.Write(contents)
	}
}

// String returns the server name and its running environment or an empty string
// of the given server name is empty.
func (m *modRevision) String() string {
	if name := m.options.ServerName; name != "" {
		if env := m.options.Env; env != "" {
			name += fmt.Sprintf(" (%s)", env)
		}

		if m.buildRevision != "" && m.buildTime != "" {
			buildTitle := ">>>> build" // if we ever want an emoji, there is one: \U0001f4bb
			tab := strings.Repeat(" ", len(buildTitle))
			name += fmt.Sprintf("\n\n%[1]s\n%srevision        %s\n[1]sbuildtime       %s\n[1]sdeveloper       %s", tab,
				buildTitle, m.buildRevision, m.buildTime, m.options.Developer)
		}

		return name
	}

	return ""
}
