package cmd

import (
	"context"
	"fmt"

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
		Use:   "jarbuild [module]",
		Short: "Build Java JAR files",
		Long: `Build Java JAR files from source code.
Pass a module name to build that JAR, use --all to build every JAR, or run
without arguments to pick a module interactively.

Examples:
  adb jarbuild reindexer
  adb jarbuild --all`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			runner, err := docker.NewRunner()
			if err != nil {
				return fmt.Errorf("initialize docker: %w", err)
			}
			defer runner.Close()

			builder := jar.NewBuilder(jar.BuildConfig{
				AspenCloneDir:   cfg.AspenCloneDir,
				JavaImage:       cfg.JavaBuildImage,
				SharedLibsPath:  cfg.JavaSharedLibsPath,
				ExcludePatterns: cfg.ExcludedJarPatterns,
			}, runner)

			ctx := context.Background()

			if all {
				return builder.BuildAll(ctx)
			}
			if len(args) == 1 {
				return buildNamedJar(ctx, builder, args[0])
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

	return buildNamedJar(ctx, builder, names[idx])
}

func buildNamedJar(ctx context.Context, builder *jar.Builder, name string) error {
	module, err := jar.FindModule(cfg.CodeDir(), name)
	if err != nil {
		return err
	}
	return builder.Build(ctx, *module)
}
