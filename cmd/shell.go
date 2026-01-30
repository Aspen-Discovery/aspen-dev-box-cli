package cmd

import (
	"context"
	"fmt"

	"adb/pkg/config"
	"adb/pkg/docker"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(ShellCommand())
}

func ShellCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "shell",
		Short: "Open a shell inside the main container",
		Long: `Open an interactive shell inside the main container.
This command opens a bash shell in the main container with the working directory set to the Aspen Discovery installation.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			runner, err := docker.NewRunner()
			if err != nil {
				return fmt.Errorf("initialize docker: %w", err)
			}
			defer runner.Close()

			return runner.ExecInteractive(context.Background(), docker.ExecConfig{
				Container:  config.GetMainContainerName(),
				Cmd:        []string{"/bin/bash"},
				WorkingDir: config.GetMainContainerWorkDir(),
			})
		},
	}
}
