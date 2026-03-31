package cmd

import (
	"context"
	"fmt"
	"os"

	"adb/pkg/config"
	"adb/pkg/docker"

	"github.com/spf13/cobra"
)

var cfg *config.Config

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "adb",
	Short: "Aspen Dev Box CLI",
	Long: `Aspen Dev Box CLI is a command-line tool for managing the Aspen Discovery development environment.

This tool provides a comprehensive set of commands to:
- Manage Docker containers and services
- Build and compile code
- Access logs and databases
- Install shell completions
- And more...

For detailed information about each command, use 'adb help <command>'.`,
	// Enable shell completion
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd:   true,
		DisableNoDescFlag:   false,
		DisableDescriptions: false,
	},
	// Don't show usage on errors
	SilenceUsage: true,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	var err error
	cfg, err = config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func resolveContainerConfig(runner *docker.SDKRunner) {
	env, err := runner.ContainerEnv(context.Background(), cfg.MainContainerName)
	if err != nil {
		return
	}
	cfg.ApplyContainerEnv(env)
}
