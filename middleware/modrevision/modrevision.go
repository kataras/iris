// Package modrevision serves the build information of the running binary
// (module version, VCS revision, build time) on a route.
//
// Its entry point is New(Options), which returns a Handler.
package modrevision

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/core/pkg/buildinfo"
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
	// CloudDetection is used to detect and log if the server is running on a cloud provider.
	// Defaults to false.
	CloudDetection bool
}

// New returns an Iris Handler which renders
// the server name (env), build information (if available)
// and an OK message. The handler displays simple debug information such as build commit id and time.
// It does NOT render information about the Go language itself or any operating system confgiuration
// for security reasons.
//
// Example Code:
//
//	app.Get("/health", modrevision.New(modrevision.Options{
//	 ServerName:   "Iris Server",
//	 Env:          "development",
//	 Developer:    "kataras",
//	 TimeLocation: time.FixedZone("Greece/Athens", 7200),
//	 UnixTime:     true,
//	 CloudDetection: true,
//	}))
func New(opts Options) context.Handler {
	buildTime, buildRevision := buildinfo.BuildTime, buildinfo.BuildRevision
	if opts.UnixTime {
		if t, err := time.Parse(time.RFC3339, buildTime); err == nil {
			buildTime = fmt.Sprintf("%d", t.Unix())
		}
	} else if opts.TimeLocation != nil {
		if t, err := time.Parse(time.RFC3339, buildTime); err == nil {
			buildTime = t.In(opts.TimeLocation).String()
		}
	}

	var buildInfo string
	if buildInfo = opts.ServerName; buildInfo != "" {
		if env := opts.Env; env != "" {
			buildInfo += fmt.Sprintf(" (%s)", env)
		}
	}

	if buildRevision != "" && buildTime != "" {
		buildTitle := ">>>> build"
		tab := strings.Repeat(" ", len(buildTitle))
		buildInfo += fmt.Sprintf("\n\n%s\n%[2]srevision        %[3]s\n%[2]sbuildtime       %[4]s\n%[2]sdeveloper       %[5]s",
			buildTitle, tab, buildRevision, buildTime, opts.Developer)
	}

	if opts.CloudDetection {
		if cloudContainer := buildinfo.DetectCloudContainer(); cloudContainer != nil {
			if jsonInfo, err := json.MarshalIndent(cloudContainer, "  ", "  "); err == nil {
				buildInfo += fmt.Sprintf("\n\n>>>> cloud container\n%s", string(jsonInfo))
			}
		}
	}

	contents := []byte(buildInfo)
	if len(contents) > 0 {
		contents = append(contents, []byte("\n\nOK")...)
	} else {
		contents = []byte("OK")
	}

	return func(ctx *context.Context) {
		ctx.Write(contents)
	}
}
