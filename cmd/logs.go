package cmd

import (
	"fmt"

	"adb/pkg/docker"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(LogsCommand())
}

func LogsCommand() *cobra.Command {
	var includeIndexing bool
	var follow bool
	var containerOutput bool

	cmd := &cobra.Command{
		Use:   "logs",
		Short: "View site logs or the container's own output",
		Long: `Tail the aspen site logs (/var/log/aspen-discovery/<SITE_NAME>/) from the
main container. --include-indexing adds the indexing logs. --container shows
the container's own stdout instead: the entrypoint output, boot progress and
the "Aspen dev box ready" banner.

Examples:
  adb logs -f
  adb logs -f -i
  adb logs -c -f`,
		RunE: func(cmd *cobra.Command, args []string) error {
			runner, err := docker.NewRunner()
			if err != nil {
				return fmt.Errorf("initialize docker: %w", err)
			}
			defer runner.Close()

			if containerOutput {
				return runner.StreamContainerLogs(cmd.Context(), cfg.MainContainerName(), follow)
			}

			resolveContainerConfig(runner)
			return runner.ExecInteractive(cmd.Context(), docker.ExecConfig{
				Container: cfg.MainContainerName(),
				Cmd:       []string{"/bin/bash", "-c", siteLogsCommand(includeIndexing, follow)},
			})
		},
	}

	cmd.Flags().BoolVarP(&includeIndexing, "include-indexing", "i", false, "Include indexing logs")
	cmd.Flags().BoolVarP(&follow, "follow", "f", false, "Follow logs in real-time")
	cmd.Flags().BoolVarP(&containerOutput, "container", "c", false, "Show the container's own output (entrypoint, boot banner) instead of the site logs")

	return cmd
}

func siteLogsCommand(includeIndexing, follow bool) string {
	logsPattern := "./*"
	if includeIndexing {
		logsPattern += " ./logs/*"
	}
	tailCmd := "tail"
	if follow {
		tailCmd += " -f"
	}
	return fmt.Sprintf("cd %s && %s %s", cfg.LogPath, tailCmd, logsPattern)
}
