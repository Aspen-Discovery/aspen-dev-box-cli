package cmd

import (
	"adb/pkg/docker"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(StopCommand(), StartCommand(), RestartCommand())
}

func stackCompose() *docker.Compose {
	return docker.NewCompose(docker.ComposeConfig{Project: cfg.StackName})
}

func StopCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "stop [service...]",
		Short: "Stop the stack without removing it",
		Long: `Stop the stack's containers. Unlike adb down, the containers, the database
and the generated ILS SQL are kept, so adb start brings everything back as it
was. Pass service names to stop only those.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return stackCompose().Stop(cmd.Context(), args...)
		},
	}
}

func StartCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "start [service...]",
		Short: "Start a stopped stack",
		Long: `Start the containers of a stack that was stopped with adb stop. Pass service
names to start only those.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return stackCompose().Start(cmd.Context(), args...)
		},
	}
}

func RestartCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "restart [service...]",
		Short: "Restart the stack, re-running the entrypoints",
		Long: `Restart the stack's containers in place. The database is kept and the aspen
entrypoint runs again, which picks up changes to dockerrun.sh or the site
config without a full adb down / adb up. Pass service names to restart only
those, e.g. adb restart aspen-dev-box.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return stackCompose().Restart(cmd.Context(), args...)
		},
	}
}
