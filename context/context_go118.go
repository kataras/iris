//go:build go1.18
// +build go1.18

package context

import "runtime/debug"

func init() {
	if info, ok := debug.ReadBuildInfo(); ok {
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
