package cmd

import (
	"adb/pkg/docker"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(DownCommand())
}

func DownCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "down",
		Short: "Bring down the Docker Compose project",
		Long: `Bring down the Docker Compose project and remove orphaned containers.
This command stops and removes all containers defined in the docker-compose file.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			compose := docker.NewCompose(docker.ComposeConfig{
				Files: []string{cfg.DefaultComposeFilePath()},
			})
			return compose.Down(cmd.Context())
		},
	}
}
