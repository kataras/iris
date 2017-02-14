package cli

import (
	goflags "flag"
	"strings"
)

type (
	// Action the command's listener
	Action func(Flags) error

	// Commands just a slice of commands
	Commands []*Cmd

	// Cmd is the command struct which contains the command's Name, description, any flags, any subcommands and mostly its action listener
	Cmd struct {
		Name        string
		Description string
		// Flags are not the arguments was given by the user, but the flags that developer sets to this command
		Flags       Flags
		action      Action
		Subcommands Commands
		flagset     *goflags.FlagSet
	}
)

var errNoAction = "No action given for %s. Please get help via --help"

// DefaultAction is the default action where no command's action is defined
func DefaultAction(cmdName string) Action {
	return func(a Flags) error { Printf(errNoAction, cmdName); return nil }
}

// Command builds and returns a new *Cmd, no app-relative.
// Note that the same command can be used by multiple applications.
// example:
// createCmd := cli.Command("create", "creates a project to a given directory").
// 	Flag("offline", false, "set to true to disable the packages download on each create command").
// 	Flag("dir", workingDir, "$GOPATH/src/$dir the directory to install the sample package").
// 	Flag("type", "basic", "creates a project based on the -t package. Currently, available types are 'basic' & 'static'").
// 	Action(create)
//
// 	app = cli.NewApp("iris", "Command line tool for Iris web framework", Version)
//  app.Command(createCmd) // registers the command to this app
//
// returns itself
func Command(name string, description string) *Cmd {
	name = strings.Replace(name, "-", "", -1) //removes all - if present, --help -> help
	fset := goflags.NewFlagSet(name, goflags.PanicOnError)
	return &Cmd{Name: name, Description: description, Flags: Flags{}, action: DefaultAction(name), flagset: fset}
}

// Subcommand adds a child command (subcommand)
//
// returns itself
func (c *Cmd) Subcommand(subCommand *Cmd) *Cmd {
	c.Subcommands = append(c.Subcommands, subCommand)
	return c
}

// Flag sets a flag to the command
// example:
// createCmd := cli.Command("create", "creates a project to a given directory").
// 	Flag("dir", "the default value/path is setted here", "$GOPATH/src/$dir the directory to install the sample package").Flag(...).Action(...)
//
// returns the Cmd itself
func (c *Cmd) Flag(name string, defaultValue interface{}, usage string) *Cmd {
	if c.Flags == nil {
		c.Flags = Flags{}
	}
	valPointer := requestFlagValue(c.flagset, name, defaultValue, usage)

	newFlag := &Flag{name, defaultValue, usage, valPointer, c.flagset}
	c.Flags = append(c.Flags, newFlag)
	return c
}

// Action is the most important function
// declares a command's action/listener
// example:
// createCmd := cli.Command("create", "creates a project to a given directory").
// 	Flag("dir", workingDir, "the directory to install the sample package").
// 	Action(func(flags cli.Flags){ /* here the action */ })
//
// returns itself
func (c *Cmd) Action(action Action) *Cmd {
	c.action = action
	return c
}

// Execute calls the Action of the command and the subcommands
// returns true if this command has been executed successfully
func (c *Cmd) Execute(parentflagset *goflags.FlagSet) bool {
	var index = -1
	// check if this command has been called from app's arguments
	for idx, a := range parentflagset.Args() {
		if c.Name == a {
			index = idx + 1
		}
	}

	// this command hasn't been called from the user
	if index == -1 {
		return false
	}

	//check if it was help sub command
	wasHelp := parentflagset.Arg(1) == "-h"

	if wasHelp {
		// global -help, --help, help, -h now shows all the help for each subcommand and subflags
		Printf("Please use global '-help' or 'help' without quotes, instead.")
		return true
	}

	if !c.flagset.Parsed() {

		if err := c.flagset.Parse(parentflagset.Args()[index:]); err != nil {
			panic("Panic on command.Execute: " + err.Error())
		}
	}

	if err := c.Flags.Validate(); err == nil {
		c.action(c.Flags)

		for idx := range c.Subcommands {
			if c.Subcommands[idx].Execute(c.flagset) {
				break
			}

		}
	} else {
		Printf(err.Error())
	}
	return true

}
