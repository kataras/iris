package cli

import (
	goflags "flag"
	"fmt"
	"os"
	"text/template"
)

// Output the common output of the cli app, defaults to os.Stdout
var Output = os.Stdout // the output is the same for all Apps, atm.

// Printf like  fmt.Printf but prints on the Output
func Printf(format string, args ...interface{}) {
	fmt.Fprintf(Output, format, args...)
}

// HelpMe receives an app and prints the help message for this app
func HelpMe(app App) {
	tmplStr := appTmpl
	tmpl, err := template.New(app.Name).Parse(tmplStr)
	if err != nil {
		panic("Panic: " + err.Error())
	}

	tmpl.Execute(Output, app)

}

// App is the container of all commands, subcommands, flags and your application's name and description
// it builds the full help message and other things useful for your app
// example of initialize an cli.App in the init():
// var app *cli.App
// init the cli app
// func init(){
// app = cli.NewApp("iris", "Command line tool for Iris web framework", Version)
// // version command
// app.Command(cli.Command("version", "\t      prints your iris version").
//   Action(func(cli.Flags) error { app.Printf("%s", iris.Version); return nil }))
//
// // create command
// createCmd := cli.Command("create", "creates & runs a project based on the iris-contrib/examples, to a given directory").
// 	Flag("dir", workingDir, "the directory to install the sample package").
// 	Flag("type", "basic", "creates a project based on the -t package. Currently, available types are 'basic' & 'static'").
// 	Action(create)
//
// // run command
// runAndWatchCmd := cli.Command("run", "runs and reload on source code changes, example: iris run main.go").Action(runAndWatch)
//
// // register the commands
// app.Command(createCmd)
// app.Command(runAndWatchCmd)
// }
// func main() {
// 	// run the application
// 	app.Run(func(cli.Flags) error {
// 		return nil
// 	})
// }
type App struct {
	Name        string
	Description string
	Version     string
	Commands    Commands
	Flags       Flags
}

// NewApp creates and returns a new cli App instance
//
// example:
// app := cli.NewApp("iris", "Command line tool for Iris web framework", Version)
func NewApp(name string, description string, version string) *App {
	return &App{name, description, version, nil, nil}
}

// Command adds a  command to the app
func (a *App) Command(cmd *Cmd) *App {
	if a.Commands == nil {
		a.Commands = Commands{}
	}

	a.Commands = append(a.Commands, cmd)
	return a
}

// Flag creates a new flag based on name, defaultValue (if empty then the flag is required) and a usage
func (a *App) Flag(name string, defaultValue interface{}, usage string) *App {
	if a.Flags == nil {
		a.Flags = Flags{}
	}

	a.Flags = append(a.Flags, &Flag{name, defaultValue, usage, nil, nil})
	return a
}

func (a App) help() {
	HelpMe(a)
	os.Exit(1)
}

// HasCommands returns true if the app has registered commands, otherwise false
func (a App) HasCommands() bool {
	return a.Commands != nil && len(a.Commands) > 0
}

// HasFlags returns true if the app has its own global flags, otherwise false
func (a App) HasFlags() bool {
	return a.Flags != nil && len(a.Flags) > 0
}

// Run executes the App, must be called last
// it receives an action too, like the commands but this action can be empty
// but if this action returns an error then the execution panics.
// Its common usage is when you need to 'monitor' the application's commands flags
// and check if a command should execute based its name or no (the app's command is a flag to the app's action function)
func (a App) Run(appAction Action) {

	flagset := goflags.NewFlagSet(a.Name, goflags.PanicOnError)
	flagset.SetOutput(Output)

	if a.Flags != nil {

		//now, get the args and set the flags
		for idx, arg := range a.Flags {
			valPointer := requestFlagValue(flagset, arg.Name, arg.Default, arg.Usage)
			a.Flags[idx].Value = valPointer
		}
	}

	if len(os.Args) <= 1 {
		a.help()
	}

	// if help argument/flag is passed
	if len(os.Args) > 1 && (os.Args[1] == "help" || os.Args[1] == "-help" || os.Args[1] == "--help") || os.Args[1] == "-h" {
		a.help()

	}
	// if flag parsing failed, yes we check it after --help.
	if err := flagset.Parse(os.Args[1:]); err != nil {
		a.help()
	}

	//first we check for commands, if any command executed then  app action should NOT be executed

	var ok = false
	for idx := range a.Commands {
		if ok = a.Commands[idx].Execute(flagset); ok {
			break
		}
	}

	if !ok {
		if err := a.Flags.Validate(); err == nil {
			if err = appAction(a.Flags); err == nil {
				ok = true
			} else {
				Printf(err.Error())
				return
			}
		} else {
			Printf(err.Error())
			return
		}
	}

	if !ok {
		a.help()
	}

}

// Printf just like fmt.Printf but prints to the application's output, which is the global Output
func (a *App) Printf(format string, args ...interface{}) {
	Printf(format, args...)
}

var appTmpl = `NAME:
   {{.Name}} - {{.Description}}

USAGE:
{{- if .HasFlags}}
   {{.Name}} [global arguments...]
{{end -}}
{{ if .HasCommands }}
   {{.Name}} command [arguments...]
{{ end }}
VERSION:
   {{.Version}}

{{ if.HasFlags}}
GLOBAL ARGUMENTS:
{{ range $idx,$flag := .Flags }}
   -{{$flag.Alias }}        {{$flag.Usage}} (default '{{$flag.Default}}')
{{ end }}
{{end -}}
{{ if .HasCommands }}
COMMANDS:
{{ range $index, $cmd := .Commands }}
   {{$cmd.Name }} {{$cmd.Flags.ToString}}        {{$cmd.Description}}
     {{ range $index, $subcmd := .Subcommands }}
     {{$subcmd.Name}}        {{$subcmd.Description}}
	 {{ end }}
     {{ range $index, $subflag := .Flags }}
      -{{$subflag.Alias }}        {{$subflag.Usage}} (default '{{$subflag.Default}}')
	 {{ end }}
{{ end }}
{{ end }}
`
