package cmd

import (
	"fmt"

	"adb/pkg/docker"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(CronCommand())
}

func CronCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "cron start|stop|status",
		Short: "Control aspen's cron inside the running container",
		Long: `Start or stop the cron daemon that runs aspen's scheduled jobs (reindexer,
ILS exports, cleanup) in the main container. Stop it while stepping through a
job with adb debug so the scheduler does not launch a second copy. adb up
--no-cron boots the box with cron already stopped.

Examples:
  adb cron stop
  adb cron status
  adb cron start`,
		Args:      cobra.ExactArgs(1),
		ValidArgs: []string{"start", "stop", "status"},
		RunE: func(cmd *cobra.Command, args []string) error {
			action := args[0]
			validAction := action == "start" || action == "stop" || action == "status"
			if !validAction {
				return fmt.Errorf("expected start, stop or status, got %q", action)
			}
			runner, err := docker.NewRunner()
			if err != nil {
				return fmt.Errorf("initialize docker: %w", err)
			}
			defer runner.Close()
			return runContainerScript(cmd.Context(), runner, "service cron \"$1\"", action, nil)
		},
	}
}
