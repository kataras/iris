// +build !windows

package main

import (
	"os/exec"
)

func sh(script string, args ...string) error {
	return (&platform{"bin/sh"}).attach("debug", append([]string{script}, args...)...)
}

func installDotnet(b bundle) error {
	return sh("./scripts/dotnet-install.sh", b.parseArguments()...)
}
