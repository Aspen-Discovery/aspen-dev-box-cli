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
	rootCmd.AddCommand(DownCommand())
}

func DownCommand() *cobra.Command {
	var all bool
	var keepData bool
	cmd := &cobra.Command{
		Use:   "down",
		Short: "Bring down the Docker Compose project",
		Long: `Bring down the Docker Compose project and remove orphaned containers.
By default the database and Solr volumes and the generated ILS SQL go too, so
the next adb up starts from a fresh database. --keep-data keeps them: the
next adb up of the same stack reuses the database and index instead of
re-initialising. The aspen proxy is stopped along with the last proxied
stack. --all brings down every aspen stack (any worktree or stack name) and
the proxy.

Examples:
  adb down
  adb down --keep-data
  adb down --all`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if all {
				return downAll(ctx, keepData)
			}
			if err := downStack(ctx, cfg.StackName, keepData); err != nil {
				return err
			}
			return maybeDownProxy(ctx)
		},
	}
	cmd.Flags().BoolVar(&all, "all", false, "Bring down every aspen stack and the proxy")
	cmd.Flags().BoolVar(&keepData, "keep-data", false, "Keep the database and Solr volumes for the next adb up")
	return cmd
}

func downStack(ctx context.Context, project string, keepData bool) error {
	compose := docker.NewCompose(docker.ComposeConfig{
		Project: project,
		Files:   []string{cfg.ComposeFilePath(config.DefaultComposeFile)},
	})
	if keepData {
		return downKeepingData(ctx, compose, project)
	}
	if err := compose.Down(ctx); err != nil {
		return err
	}
	runner, err := docker.NewRunner()
	if err != nil {
		return err
	}
	defer runner.Close()
	if err := runner.RemoveProjectVolumes(ctx, project); err != nil {
		return err
	}
	if err := os.Remove(cfg.ILSSQLPath(project)); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove ils sql: %w", err)
	}
	return nil
}

func downKeepingData(ctx context.Context, compose *docker.Compose, project string) error {
	if err := compose.DownKeepingVolumes(ctx); err != nil {
		return err
	}
	fmt.Printf("Kept the data volumes of %s; adb up reuses them\n", project)
	return nil
}

func downAll(ctx context.Context, keepData bool) error {
	runner, err := docker.NewRunner()
	if err != nil {
		return err
	}
	defer runner.Close()

	projects, err := runner.ComposeProjects(ctx, cfg.MainContainerService)
	if err != nil {
		return err
	}
	for _, project := range projects {
		fmt.Printf("Bringing down %s\n", project)
		if err := downStack(ctx, project, keepData); err != nil {
			return err
		}
	}
	return maybeDownProxy(ctx)
}
