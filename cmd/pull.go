package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"adb/pkg/config"
	"adb/pkg/docker"

	"github.com/compose-spec/compose-go/loader"
	"github.com/compose-spec/compose-go/template"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(PullCommand())
}

func PullCommand() *cobra.Command {
	var evergreen bool

	cmd := &cobra.Command{
		Use:   "pull",
		Short: "Pull every image the dev box uses",
		Long: `Pull the images referenced by all compose files (the stack, every overlay and
the proxy) plus the JDK and less images used by adb jarbuild and adb
compilecss, so a later adb up needs no downloads. The Evergreen ILS image is
large and only pulled with --evergreen.

Examples:
  adb pull
  adb pull --evergreen`,
		RunE: func(cmd *cobra.Command, args []string) error {
			setupSolrImage(cfg.AspenCloneDir)

			files, err := pullableComposeFiles(evergreen)
			if err != nil {
				return err
			}
			images, err := imagesFromComposeFiles(files)
			if err != nil {
				return err
			}
			images = append(images, cfg.JavaBuildImage, cfg.LessImage)

			runner, err := docker.NewRunner()
			if err != nil {
				return fmt.Errorf("initialize docker: %w", err)
			}
			defer runner.Close()
			return pullImages(cmd.Context(), runner, images)
		},
	}

	cmd.Flags().BoolVarP(&evergreen, "evergreen", "e", false, "Also pull the Evergreen ILS image")
	return cmd
}

func pullableComposeFiles(evergreen bool) ([]string, error) {
	files, err := filepath.Glob(filepath.Join(cfg.ProjectsDir, "compose", "*.yml"))
	if err != nil {
		return nil, err
	}
	var selected []string
	for _, f := range files {
		skipEvergreen := !evergreen && filepath.Base(f) == config.EvergreenComposeFile
		if skipEvergreen {
			continue
		}
		selected = append(selected, f)
	}
	return append(selected, cfg.ProxyComposeFilePath()), nil
}

func imagesFromComposeFiles(files []string) ([]string, error) {
	seen := map[string]bool{}
	var images []string
	for _, file := range files {
		fileImages, err := imagesFromComposeFile(file)
		if err != nil {
			return nil, err
		}
		for _, image := range fileImages {
			if seen[image] {
				continue
			}
			seen[image] = true
			images = append(images, image)
		}
	}
	sort.Strings(images)
	return images, nil
}

func imagesFromComposeFile(file string) ([]string, error) {
	content, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", file, err)
	}
	loadedConfig, err := loader.ParseYAML(content)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", file, err)
	}
	services, ok := loadedConfig["services"].(map[string]interface{})
	if !ok {
		return nil, nil
	}

	var images []string
	for _, service := range services {
		serviceMap, ok := service.(map[string]interface{})
		if !ok {
			continue
		}
		imageName, ok := serviceMap["image"].(string)
		if !ok {
			continue
		}
		imageName, err = template.Substitute(imageName, os.LookupEnv)
		if err != nil {
			return nil, fmt.Errorf("resolve image in %s: %w", file, err)
		}
		images = append(images, imageName)
	}
	return images, nil
}

func pullImages(ctx context.Context, runner *docker.SDKRunner, images []string) error {
	fmt.Printf("Pulling %d images: %s\n", len(images), strings.Join(images, ", "))
	for _, image := range images {
		fmt.Printf("\nPulling %s\n", image)
		if err := runner.Pull(ctx, image); err != nil {
			return fmt.Errorf("pull %s: %w", image, err)
		}
	}
	return nil
}
