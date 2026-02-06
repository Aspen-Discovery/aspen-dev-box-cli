package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"adb/pkg/compose"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(PullCommand())
}
func PullCommand() *cobra.Command {
	var debugging bool
	var dbgui bool

	cmd := &cobra.Command{
		Use:   "pull",
		Short: "Pull Docker images for selected compose files",
		Long: `Pull Docker images defined in the selected docker-compose files.
This command pulls images only from the compose files that match the provided flags,
similar to how 'adb up' selects which services to start.

Examples:
  adb pull              # Pull base images only
  adb pull --dbgui      # Pull base + phpmyadmin images
  adb pull -g -b        # Pull base + debug + phpmyadmin images`,
		Run: func(cmd *cobra.Command, args []string) {
			// Get compose files based on flags
			composeFiles, err := compose.GetComposeFiles(compose.Options{
				Debugging: debugging,
				DBGui:     dbgui,
			})
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}

			// Build docker compose command with all selected files
			commandArgs := []string{"compose"}
			for _, file := range composeFiles {
				commandArgs = append(commandArgs, "-f", file)
			}
			commandArgs = append(commandArgs, "pull")

			// Execute docker compose pull
			command := exec.Command("docker", commandArgs...)
			command.Stdout = os.Stdout
			command.Stderr = os.Stderr

			if err := command.Run(); err != nil {
				fmt.Printf("Error pulling images: %v\n", err)
				os.Exit(1)
			}
		},
	}

	cmd.Flags().BoolVarP(&debugging, "debugging", "g", false, "Pull images for debugging compose file")
	cmd.Flags().BoolVarP(&dbgui, "dbgui", "b", false, "Pull images for dbgui compose file (phpmyadmin)")

	return cmd
}
