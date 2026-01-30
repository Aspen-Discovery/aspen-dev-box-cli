package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"adb/pkg/config"
	"adb/pkg/docker"

	"github.com/compose-spec/compose-go/loader"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(UpCommand())
}

func UpCommand() *cobra.Command {
	var detached bool
	var debugging bool
	var dbgui bool
	var pullUpdated bool
	var kohaStack string
	var ils string

	cmd := &cobra.Command{
		Use:   "up",
		Short: "Bring up the Docker Compose project",
		Long: `Bring up the Docker Compose project with optional configurations.
You can run in detached mode, with debugging enabled, or with the database GUI.
You can also select which ILS to use (koha or evergreen).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			files := []string{config.GetDefaultComposeFile()}

			if debugging {
				files = append(files, config.GetDebugComposeFile())
			}

			if dbgui {
				files = append(files, config.GetDBGUIComposeFile())
			}

			aspenDocker := config.GetProjectsDir()

			ilsFile, err := getILSComposeFile(aspenDocker, ils, kohaStack)
			if err != nil {
				return err
			}
			files = append(files, ilsFile)

			if pullUpdated {
				if err := pullImagesFromFiles(files); err != nil {
					return err
				}
			}

			compose := docker.NewCompose(docker.ComposeConfig{
				Files:    files,
				Detached: detached,
			})

			return compose.Up()
		},
	}

	cmd.Flags().BoolVarP(&detached, "detached", "d", false, "Run in detached mode")
	cmd.Flags().BoolVarP(&debugging, "debugging", "g", false, "Run with debugging compose file")
	cmd.Flags().BoolVarP(&dbgui, "dbgui", "b", false, "Run with dbgui compose file")
	cmd.Flags().BoolVarP(&pullUpdated, "pull", "p", false, "Pull the images for the project only if they have been updated")
	cmd.Flags().StringVarP(&kohaStack, "koha-stack", "k", "", "Specify the Koha stack to connect to (default: kohadev)")
	cmd.Flags().StringVarP(&ils, "ils", "i", "koha", "Select ILS to use (koha|evergreen)")

	return cmd
}

func getILSComposeFile(aspenDocker, ils, kohaStack string) (string, error) {
	switch ils {
	case "koha":
		if kohaStack != "" {
			os.Setenv("KOHA_STACK", kohaStack)
		}
		path := filepath.Join(aspenDocker, "docker-compose.koha.yml")
		if _, err := os.Stat(path); err != nil {
			return "", fmt.Errorf("Koha override file not found at %s", path)
		}
		return path, nil

	case "evergreen":
		path := filepath.Join(aspenDocker, "docker-compose.evergreen.yml")
		if _, err := os.Stat(path); err != nil {
			return "", fmt.Errorf("Evergreen override file not found at %s", path)
		}
		return path, nil

	default:
		return "", fmt.Errorf("unsupported ILS '%s'. Supported values: koha, evergreen", ils)
	}
}

func pullImagesFromFiles(files []string) error {
	runner, err := docker.NewRunner()
	if err != nil {
		return fmt.Errorf("initialize docker: %w", err)
	}
	defer runner.Close()

	ctx := context.Background()

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("read %s: %w", file, err)
		}

		loadedConfig, err := loader.ParseYAML(content)
		if err != nil {
			return fmt.Errorf("parse %s: %w", file, err)
		}

		services, ok := loadedConfig["services"].(map[string]interface{})
		if !ok {
			continue
		}

		for _, service := range services {
			serviceMap, ok := service.(map[string]interface{})
			if !ok {
				continue
			}

			imageName, ok := serviceMap["image"].(string)
			if !ok {
				continue
			}

			fmt.Printf("Pulling image: %s\n", imageName)
			if err := runner.Pull(ctx, imageName); err != nil {
				return fmt.Errorf("pull %s: %w", imageName, err)
			}
		}
	}

	return nil
}
