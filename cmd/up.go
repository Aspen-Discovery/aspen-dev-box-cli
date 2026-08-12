package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"adb/pkg/config"
	"adb/pkg/docker"
	"adb/pkg/ils"

	"github.com/compose-spec/compose-go/loader"
	"github.com/compose-spec/compose-go/template"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(UpCommand())
}

func UpCommand() *cobra.Command {
	var detached bool
	var debugging bool
	var javaDebug bool
	var dbgui bool
	var pullUpdated bool
	var kohaStack string
	var ilsFlag string
	var pluginsPath string
	var plugins bool
	var noProxy bool
	var host string

	cmd := &cobra.Command{
		Use:   "up",
		Short: "Bring up the Docker Compose project",
		Long: `Bring up the Docker Compose project with optional configurations.
You can run in detached mode, with debugging enabled, or with the database GUI.
The --ils flag accepts a preset name (koha, evergreen, ...), a path to a custom
YAML config, or "none" to skip ILS setup entirely.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			files := []string{cfg.ComposeFilePath(config.DefaultComposeFile)}

			if debugging {
				files = append(files, cfg.ComposeFilePath(config.DebugComposeFile))
			}

			if javaDebug {
				files = append(files, cfg.ComposeFilePath(config.JavaDebugComposeFile))
			}

			if dbgui {
				files = append(files, cfg.ComposeFilePath(config.DBGUIComposeFile))
			}

			ilsFiles, err := setupILS(ilsFlag, kohaStack)
			if err != nil {
				return err
			}
			files = append(files, ilsFiles...)

			pluginFiles, err := setupPlugins(plugins, pluginsPath)
			if err != nil {
				return err
			}
			files = append(files, pluginFiles...)

			proxyFiles, err := setupProxy(ctx, noProxy, host)
			if err != nil {
				return err
			}
			files = append(files, proxyFiles...)

			dbguiFiles, err := setupProxiedDBGUI(dbgui, len(proxyFiles) > 0)
			if err != nil {
				return err
			}
			files = append(files, dbguiFiles...)

			setupSolrImage(cfg.AspenCloneDir)

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
	cmd.Flags().BoolVarP(&javaDebug, "java-debug", "j", false, "Expose JDWP port 5005 and mount debug.sh for java debugging")
	cmd.Flags().BoolVarP(&dbgui, "dbgui", "b", false, "Run with dbgui compose file")
	cmd.Flags().BoolVarP(&pullUpdated, "pull", "p", false, "Pull the images for the project only if they have been updated")
	cmd.Flags().StringVarP(&kohaStack, "koha-stack", "k", "", "Koha stack to connect to (default: kohadev)")
	cmd.Flags().StringVarP(&ilsFlag, "ils", "i", "koha", "ILS preset name, path to YAML config, or 'none'")
	cmd.Flags().BoolVar(&plugins, "plugins", false, "Mount a plugins dir into the container and enable aspen plugin loading")
	cmd.Flags().StringVar(&pluginsPath, "plugins-path", "", "Host path of plugins dir (default: $ASPEN_PLUGINS or $ASPEN_DOCKER/plugins)")
	cmd.Flags().BoolVar(&noProxy, "no-proxy", false, "Bind host ports even when the aspen proxy is running")
	cmd.Flags().StringVar(&host, "host", "", "Hostname to serve when proxied (default: <stack>.localhost)")

	return cmd
}

func setupProxy(ctx context.Context, disabled bool, host string) ([]string, error) {
	if disabled && host != "" {
		return nil, fmt.Errorf("--host conflicts with --no-proxy")
	}
	if disabled {
		return nil, nil
	}
	port, err := ensureProxy(ctx)
	if err != nil {
		return nil, err
	}
	if host == "" && defaultInstance() {
		host = "localhost"
	}
	if host == "" {
		host = cfg.StackName + ".localhost"
	}
	url := os.Getenv("ASPEN_URL")
	if url == "" {
		url = "http://" + host
		if port != 80 {
			url = fmt.Sprintf("%s:%d", url, port)
		}
		os.Setenv("ASPEN_URL", url)
	}
	os.Setenv("ASPEN_STACK", cfg.StackName)
	os.Setenv("ASPEN_HOST", host)
	overlay := cfg.ComposeFilePath(config.ProxyComposeFile)
	if _, err := os.Stat(overlay); err != nil {
		return nil, fmt.Errorf("proxy overlay missing: %s", overlay)
	}
	fmt.Printf("Aspen proxy detected — serving on %s\n", url)
	return []string{overlay}, nil
}

func setupProxiedDBGUI(dbgui, proxied bool) ([]string, error) {
	if !dbgui || !proxied {
		return nil, nil
	}
	overlay := cfg.ComposeFilePath(config.ProxiedDBGUIComposeFile)
	if _, err := os.Stat(overlay); err != nil {
		return nil, fmt.Errorf("proxied dbgui overlay missing: %s", overlay)
	}
	return []string{overlay}, nil
}

func defaultInstance() bool {
	return worktreeName == "" && cfg.StackName == filepath.Base(cfg.ProjectsDir)
}

func setupPlugins(enabled bool, path string) ([]string, error) {
	if !enabled {
		return nil, nil
	}
	if path != "" {
		abs, err := filepath.Abs(path)
		if err != nil {
			return nil, err
		}
		os.Setenv("ASPEN_PLUGINS", abs)
	}
	overlay := cfg.ComposeFilePath(config.PluginsComposeFile)
	if _, err := os.Stat(overlay); err != nil {
		return nil, fmt.Errorf("plugins overlay missing: %s", overlay)
	}
	return []string{overlay}, nil
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

	sqlPath := cfg.ILSSQLPath(cfg.StackName)
	if err := ilsCfg.WriteSQL(sqlPath, cfg.ProjectsDir); err != nil {
		return nil, fmt.Errorf("write ils sql: %w", err)
	}
	os.Setenv("ADB_ILS_SQL", sqlPath)

	overlays := []string{cfg.ComposeFilePath(config.ILSComposeFile)}
	if value == "koha" {
		overlays = append(overlays, cfg.ComposeFilePath(config.KohaComposeFile))
	}
	if value == "evergreen" {
		overlays = append(overlays, cfg.ComposeFilePath(config.EvergreenComposeFile))
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

			imageName, err = template.Substitute(imageName, os.LookupEnv)
			if err != nil {
				return fmt.Errorf("resolve image for %s: %w", file, err)
			}

			fmt.Printf("Pulling image: %s\n", imageName)
			if err := runner.Pull(ctx, imageName); err != nil {
				return fmt.Errorf("pull %s: %w", imageName, err)
			}
		}
	}

	return nil
}
