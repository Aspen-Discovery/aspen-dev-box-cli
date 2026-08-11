package cmd

import (
	"adb/pkg/config"
	"adb/pkg/docker"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(PullCommand())
}

func PullCommand() *cobra.Command {
	var debugging bool
	var dbgui bool
	var evergreen bool

	cmd := &cobra.Command{
		Use:   "pull",
		Short: "Pull Docker images for selected compose files",
		Long: `Pull Docker images defined in the selected docker-compose files.
This command pulls images only from the compose files that match the provided flags,
similar to how 'adb up' selects which services to start.

Examples:
  adb pull              # Pull base images only
  adb pull --dbgui      # Pull base + phpmyadmin images
  adb pull -g -b        # Pull base + debug + phpmyadmin images
  adb pull --evergreen  # Pull base + evergreen images`,
		RunE: func(cmd *cobra.Command, args []string) error {
			setupSolrImage(cfg.AspenCloneDir)

			files := []string{cfg.ComposeFilePath(config.DefaultComposeFile)}

			if debugging {
				files = append(files, cfg.ComposeFilePath(config.DebugComposeFile))
			}

			if dbgui {
				files = append(files, cfg.ComposeFilePath(config.DBGUIComposeFile))
			}

			if evergreen {
				files = append(files, cfg.ComposeFilePath(config.EvergreenComposeFile))
			}

			compose := docker.NewCompose(docker.ComposeConfig{
				Project: cfg.StackName,
				Files:   files,
			})

			return compose.Pull(cmd.Context())
		},
	}

	cmd.Flags().BoolVarP(&debugging, "debugging", "g", false, "Pull images for debugging compose file")
	cmd.Flags().BoolVarP(&dbgui, "dbgui", "b", false, "Pull images for dbgui compose file (phpmyadmin)")
	cmd.Flags().BoolVarP(&evergreen, "evergreen", "e", false, "Pull images for evergreen compose file")

	return cmd
}
