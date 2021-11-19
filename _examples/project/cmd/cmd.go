package cmd

import (
	"github.com/username/project/api"

	"github.com/spf13/cobra"
)

const defaultConfigFilename = "server.yml"

var serverConfig api.Configuration

// New returns a new CLI app.
// Build with:
// $ go build -ldflags="-s -w"
func New(buildRevision, buildTime string) *cobra.Command {
	configFile := defaultConfigFilename

	rootCmd := &cobra.Command{
		Use:                        "project",
		Short:                      "Command line interface for project.",
		Long:                       "The root command will start the HTTP server.",
		Version:                    "v0.0.1",
		SilenceErrors:              true,
		SilenceUsage:               true,
		TraverseChildren:           true,
		SuggestionsMinimumDistance: 1,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Read configuration from file before any of the commands run functions.
			return serverConfig.BindFile(configFile)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return startServer()
		},
	}

	helpTemplate := HelpTemplate{
		BuildRevision:        buildRevision,
		BuildTime:            buildTime,
		ShowGoRuntimeVersion: true,
	}
	rootCmd.SetHelpTemplate(helpTemplate.String())

	// Shared flags.
	flags := rootCmd.PersistentFlags()
	flags.StringVar(&configFile, "config", configFile, "--config=server.yml a filepath which contains the YAML config format")

	// Subcommands here.
	// rootCmd.AddCommand(...)

	return rootCmd
}

func startServer() error {
	srv := api.NewServer(serverConfig)
	return srv.Start()
}
