package cmd

import (
	"context"
	"fmt"
	"strings"

	"adb/pkg/config"
	"adb/pkg/docker"
	"adb/pkg/jar"

	fuzzyfinder "github.com/ktr0731/go-fuzzyfinder"
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

	idx, err := fuzzyfinder.Find(
		names,
		func(i int) string {
			return names[i]
		},
	)
	if err != nil {
		if err == fuzzyfinder.ErrAbort {
			return fmt.Errorf("selection cancelled")
		}
		return fmt.Errorf("fuzzy finder: %w", err)
	}

	codeDir := fmt.Sprintf("%s/code", config.GetAspenCloneDir())
	module, err := jar.FindModule(codeDir, names[idx])
	if err != nil {
		return err
	}

	return builder.Build(ctx, *module)
}
