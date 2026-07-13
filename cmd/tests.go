package cmd

import (
	"fmt"
	"os"

	"adb/pkg/docker"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(TestsCommand())
}

func TestsCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "tests [phpunit args...]",
		Short: "Run the phpunit suite in a throwaway unit-tests container",
		Long: `Run the Aspen Discovery phpunit suite in a dedicated container against its own
unit tests database. The dev database and running containers are not touched,
so the stack must already be up (adb up -d).

Extra arguments are passed through to phpunit.

Examples:
  adb tests
  adb tests --filter UserAPITests`,
		Args:               cobra.ArbitraryArgs,
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			runner, err := docker.NewRunner()
			if err != nil {
				return fmt.Errorf("initialize docker: %w", err)
			}
			defer runner.Close()

			running, err := runner.ContainerRunning(ctx, cfg.DBContainerName())
			if err != nil {
				return err
			}
			if !running {
				return fmt.Errorf("%s is not running, start the stack first with 'adb up -d'", cfg.DBContainerName())
			}

			os.Setenv("COMPOSE_IGNORE_ORPHANS", "1")
			compose := docker.NewCompose(docker.ComposeConfig{
				Project: cfg.StackName,
				Files:   []string{cfg.DefaultComposeFilePath(), cfg.TestsComposeFilePath()},
				Quiet:   true,
			})
			return compose.RunService(ctx, "unit-tests", args)
		},
	}
}
