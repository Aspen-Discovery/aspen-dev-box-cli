package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"adb/pkg/config"
	"adb/pkg/docker"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

func init() {
	rootCmd.AddCommand(PullCommand())
}

func PullCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "pull",
		Short: "Pull Docker images",
		Long: `Pull all Docker images defined in docker-compose files.
This command scans the ASPEN_DOCKER directory for docker-compose files
and pulls all images defined in their services.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			runner, err := docker.NewRunner()
			if err != nil {
				return fmt.Errorf("initialize docker: %w", err)
			}
			defer runner.Close()

			ctx := context.Background()
			projectsDir := config.GetProjectsDir()

			return filepath.Walk(projectsDir, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return fmt.Errorf("access path %q: %w", path, err)
				}

				if filepath.Ext(path) != ".yml" {
					return nil
				}

				images, err := extractImagesFromCompose(path)
				if err != nil {
					fmt.Printf("Warning: %v\n", err)
					return nil
				}

				for _, img := range images {
					fmt.Printf("Pulling image: %s\n", img)
					if err := runner.Pull(ctx, img); err != nil {
						return fmt.Errorf("pull %s: %w", img, err)
					}
				}

				return nil
			})
		},
	}
}

func extractImagesFromCompose(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	var composeFile map[string]interface{}
	if err := yaml.Unmarshal(data, &composeFile); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	services, ok := composeFile["services"].(map[interface{}]interface{})
	if !ok {
		return nil, nil
	}

	var images []string
	for _, service := range services {
		serviceMap, ok := service.(map[interface{}]interface{})
		if !ok {
			continue
		}

		if img, ok := serviceMap["image"].(string); ok {
			images = append(images, img)
		}
	}

	return images, nil
}
