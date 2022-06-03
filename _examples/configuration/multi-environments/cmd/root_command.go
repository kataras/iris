package cmd

import (
	"github.com/kataras/my-iris-app/api"

	"github.com/spf13/cobra"
)

const defaultConfigFilename = "server.dev.yml"

var (
	serverConfig api.Configuration
)

// New returns a new root command.
// Usage:
// $ my-iris-app --config=server.yml
func New() *cobra.Command {
	configFile := defaultConfigFilename

	rootCmd := &cobra.Command{
		Use:                        "my-iris-app",
		Short:                      "My Command Line App",
		Long:                       "The root command will start the HTTP server.",
		Version:                    "v0.0.1",
		SilenceErrors:              true,
		SilenceUsage:               true,
		TraverseChildren:           true,
		SuggestionsMinimumDistance: 1,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return serverConfig.BindFile(configFile)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return startServer()
		},
	}

	// Shared flags.
	flags := rootCmd.PersistentFlags()
	flags.StringVar(&configFile, "config", configFile, "--config=server.yml a filepath which contains the YAML config format")

	// Subcommands here.

	return rootCmd
}

func startServer() error {
	srv := api.NewServer(serverConfig)
	return srv.Listen()
}
