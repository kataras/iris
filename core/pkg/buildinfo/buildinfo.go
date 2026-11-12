package buildinfo

import (
	"runtime/debug"
)

var (
	// BuildRevision holds the vcs commit id information of the program's build.
	// Available at go version 1.18+
	BuildRevision string
	// BuildTime holds the vcs commit time information of the program's build.
	// Available at go version 1.18+
	BuildTime string
	// ModuleName holds the module name of the program's build.
	ModuleName string
)

func init() {
	if info, ok := debug.ReadBuildInfo(); ok {
		ModuleName = info.Path

		for _, setting := range info.Settings {
			if BuildRevision != "" && BuildTime != "" {
				break
			}

			if setting.Key == "vcs.revision" {
				BuildRevision = setting.Value
			}

			if setting.Key == "vcs.time" {
				BuildTime = setting.Value
			}
		}
	}
}
