package cmd

import (
	"context"
	"fmt"

	"adb/pkg/config"
	"adb/pkg/docker"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(DownCommand())
}

func DownCommand() *cobra.Command {
	var all bool
	cmd := &cobra.Command{
		Use:   "down",
		Short: "Bring down the Docker Compose project",
		Long: `Bring down the Docker Compose project and remove orphaned containers.
The aspen proxy is stopped along with the last proxied stack. --all brings
down every aspen stack (any worktree or stack name) and the proxy.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if all {
				return downAll(ctx)
			}
			if err := downStack(ctx, cfg.StackName); err != nil {
				return err
			}
			return maybeDownProxy(ctx)
		},
	}
	cmd.Flags().BoolVar(&all, "all", false, "Bring down every aspen stack and the proxy")
	return cmd
}

func downStack(ctx context.Context, project string) error {
	compose := docker.NewCompose(docker.ComposeConfig{
		Project: project,
		Files:   []string{cfg.ComposeFilePath(config.DefaultComposeFile)},
	})
	return compose.Down(ctx)
}

func downAll(ctx context.Context) error {
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
		if err := downStack(ctx, project); err != nil {
			return err
		}
	}
	return maybeDownProxy(ctx)
}
