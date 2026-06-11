package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"adb/pkg/docker"
	"adb/pkg/ils"

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
	var ilsFlag string

	cmd := &cobra.Command{
		Use:   "up",
		Short: "Bring up the Docker Compose project",
		Long: `Bring up the Docker Compose project with optional configurations.
You can run in detached mode, with debugging enabled, or with the database GUI.
The --ils flag accepts a preset name (koha, evergreen, ...), a path to a custom
YAML config, or "none" to skip ILS setup entirely.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			files := []string{cfg.DefaultComposeFilePath()}

			if debugging {
				files = append(files, cfg.DebugComposeFilePath())
			}

			if dbgui {
				files = append(files, cfg.DBGUIComposeFilePath())
			}

			ilsFiles, err := setupILS(ilsFlag, kohaStack)
			if err != nil {
				return err
			}
			files = append(files, ilsFiles...)

			if pullUpdated {
				if err := pullImagesFromFiles(ctx, files); err != nil {
					return err
				}
			}

			compose := docker.NewCompose(docker.ComposeConfig{
				Project:  cfg.StackName,
				Files:    files,
				Detached: detached,
			})

			return compose.Up(ctx)
		},
	}

	cmd.Flags().BoolVarP(&detached, "detached", "d", false, "Run in detached mode")
	cmd.Flags().BoolVarP(&debugging, "debugging", "g", false, "Run with debugging compose file")
	cmd.Flags().BoolVarP(&dbgui, "dbgui", "b", false, "Run with dbgui compose file")
	cmd.Flags().BoolVarP(&pullUpdated, "pull", "p", false, "Pull the images for the project only if they have been updated")
	cmd.Flags().StringVarP(&kohaStack, "koha-stack", "k", "", "Koha stack to connect to (default: kohadev)")
	cmd.Flags().StringVarP(&ilsFlag, "ils", "i", "koha", "ILS preset name, path to YAML config, or 'none'")

	return cmd
}

func setupILS(value, kohaStack string) ([]string, error) {
	if value == "" || value == "none" {
		return nil, nil
	}

	if kohaStack == "" {
		kohaStack = "kohadev"
	}
	os.Setenv("KOHA_STACK", kohaStack)

	configPath, err := ils.ResolvePath(value, filepath.Join(cfg.ProjectsDir, "ils"))
	if err != nil {
		return nil, err
	}

	ilsCfg, err := ils.Load(configPath)
	if err != nil {
		return nil, err
	}

	sqlPath := filepath.Join(cfg.ProjectsDir, ".cache", "ils-setup.sql")
	if err := ilsCfg.WriteSQL(sqlPath, cfg.ProjectsDir); err != nil {
		return nil, fmt.Errorf("write ils sql: %w", err)
	}
	os.Setenv("ADB_ILS_SQL", sqlPath)

	overlays := []string{filepath.Join(cfg.ProjectsDir, "docker-compose.ils.yml")}
	if value == "koha" {
		overlays = append(overlays, filepath.Join(cfg.ProjectsDir, "docker-compose.koha.yml"))
	}

	for _, p := range overlays {
		if _, err := os.Stat(p); err != nil {
			return nil, fmt.Errorf("compose overlay missing: %s", p)
		}
	}
	return overlays, nil
}

func pullImagesFromFiles(ctx context.Context, files []string) error {
	runner, err := docker.NewRunner()
	if err != nil {
		return fmt.Errorf("initialize docker: %w", err)
	}
	defer runner.Close()

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
