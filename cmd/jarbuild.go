package cmd

import (
	"context"
	"fmt"
	"strings"

	"adb/pkg/config"
	"adb/pkg/docker"
	"adb/pkg/jar"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(JarBuildCommand())
}

func JarBuildCommand() *cobra.Command {
	var all bool

	cmd := &cobra.Command{
		Use:   "jarbuild",
		Short: "Build Java JAR files",
		Long: `Build Java JAR files from source code.
This command can build either a single JAR file selected interactively or all JAR files at once.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			runner, err := docker.NewRunner()
			if err != nil {
				return fmt.Errorf("initialize docker: %w", err)
			}
			defer runner.Close()

			builder := jar.NewBuilder(jar.BuildConfig{
				AspenCloneDir:   config.GetAspenCloneDir(),
				JavaImage:       config.GetJavaBuildImage(),
				SharedLibsPath:  config.GetJavaSharedLibrariesPath(),
				ExcludePatterns: strings.Split(config.GetExcludedJarPatterns(), " "),
			}, runner)

			ctx := context.Background()

			if all {
				return builder.BuildAll(ctx)
			}
			return buildSingleJar(ctx, builder)
		},
	}

	cmd.Flags().BoolVarP(&all, "all", "a", false, "Build all JAR files")
	return cmd
}

func buildSingleJar(ctx context.Context, builder *jar.Builder) error {
	names, err := builder.GetModuleNames(ctx)
	if err != nil {
		return fmt.Errorf("list modules: %w", err)
	}

	prompt := promptui.Select{
		Label: "Select JAR to build",
		Items: names,
		Size:  20,
	}

	_, selected, err := prompt.Run()
	if err != nil {
		return fmt.Errorf("selection cancelled: %w", err)
	}

	codeDir := fmt.Sprintf("%s/code", config.GetAspenCloneDir())
	module, err := jar.FindModule(codeDir, selected)
	if err != nil {
		return err
	}

	return builder.Build(ctx, *module)
}
