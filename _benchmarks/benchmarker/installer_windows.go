// +build windows

package main

import "errors"

var errUnableToInstall = errors.New("unable to install")

func powershell(script string, args ...string) error {
	return (&platform{"powershell"}).attach("debug", append([]string{script}, args...)...)
}

func installDotnet(b bundle) error {
	// Note: -Channel Preview is not available with the "latest" version, so we target a specific version;
	// it's also required to do that because a lot of times the project .csproj settings are changing from version to version.
	//
	// Issue:
	// cannot be loaded because running scripts is disabled on this system
	// Solution with administrator privileges:
	// Set-ExecutionPolicy -Scope CurrentUser -ExecutionPolicy Unrestricted
	//
	// Issue: ./scripts/dotnet-install.ps1 : AuthorizationManager check failed.
	// Solution (not work):
	// Unblock-File + script
	// Solution (requires manual action):
	// Right click on the ./scripts/dotnet-install.ps1 and check the "unblock" property, save and exit the dialog.
	//
	// -ExecutionPolicy Bypass? (not tested)
	return powershell("./scripts/dotnet-install.ps1", b.parseArguments()...)
}

func installNode(b bundle) error {
	return powershell("./scripts/node-install.ps1", b.parseArguments()...)
}

func installGit(b bundle) error {
	return powershell("./scripts/git-install.ps1", b.parseArguments()...)
}
