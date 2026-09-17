package cmd

import (
	"context"
	"fmt"
	"os"

	"adb/pkg/config"
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

Extra arguments are passed through to phpunit, except --updatedb which makes
the test bootstrap apply pending database updates after the fresh schema import.

Examples:
  adb tests
  adb tests --filter UserAPITests
  adb tests --updatedb --filter UserAPITests`,
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
				Files:   []string{cfg.ComposeFilePath(config.DefaultComposeFile), cfg.ComposeFilePath(config.TestsComposeFile)},
				Quiet:   true,
			})

			existing, err := runner.ProjectVolumes(ctx, cfg.StackName)
			if err != nil {
				return err
			}
			runErr := compose.RunService(ctx, "unit-tests", args)
			if err := removeVolumesCreatedByRun(ctx, runner, cfg.StackName, existing); err != nil {
				return err
			}
			return runErr
		},
	}
}

func removeVolumesCreatedByRun(ctx context.Context, runner *docker.SDKRunner, project string, existing []string) error {
	current, err := runner.ProjectVolumes(ctx, project)
	if err != nil {
		return err
	}
	kept := make(map[string]bool, len(existing))
	for _, name := range existing {
		kept[name] = true
	}
	var created []string
	for _, name := range current {
		predatesRun := kept[name]
		if predatesRun {
			continue
		}
		created = append(created, name)
	}
	return runner.RemoveVolumes(ctx, created)
}
