package cmd

import (
	"context"
	"fmt"

	"adb/pkg/docker"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(LogsCommand())
}

func LogsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logs",
		Short: "View container logs",
		Long: `View logs from the main container.
This command allows you to view and follow logs in real-time.
You can optionally include indexing logs using the --include-indexing flag.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			includeIndexing, _ := cmd.Flags().GetBool("include-indexing")
			follow, _ := cmd.Flags().GetBool("follow")

			logsPattern := "./*"
			if includeIndexing {
				logsPattern += " ./logs/*"
			}

			tailCmd := "tail"
			if follow {
				tailCmd += " -f"
			}

			runner, err := docker.NewRunner()
			if err != nil {
				return fmt.Errorf("initialize docker: %w", err)
			}
			defer runner.Close()
			resolveContainerConfig(runner)

			shellCmd := fmt.Sprintf("cd %s && %s %s", cfg.LogPath, tailCmd, logsPattern)

			return runner.ExecInteractive(context.Background(), docker.ExecConfig{
				Container: cfg.MainContainerName,
				Cmd:       []string{"/bin/bash", "-c", shellCmd},
			})
		},
	}

	cmd.Flags().BoolP("include-indexing", "i", false, "Include indexing logs")
	cmd.Flags().BoolP("follow", "f", false, "Follow logs in real-time")

	return cmd
}
