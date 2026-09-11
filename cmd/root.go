package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"

	"adb/pkg/config"
	"adb/pkg/docker"
	"adb/pkg/worktree"

	fuzzyfinder "github.com/ktr0731/go-fuzzyfinder"

	"github.com/spf13/cobra"
)

var cfg *config.Config
var worktreeName string
var pickInteractive bool

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "adb",
	Short: "Aspen Dev Box CLI",
	Long: `Aspen Dev Box CLI is a command-line tool for managing the Aspen Discovery development environment.

This tool provides a comprehensive set of commands to:
- Manage Docker containers and services
- Build and compile code
- Access logs and databases
- Install shell completions
- And more...

For detailed information about each command, use 'adb help <command>'.`,
	// Enable shell completion
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd:   false,
		DisableNoDescFlag:   false,
		DisableDescriptions: false,
	},
	// Don't show usage on errors
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if worktreeName == "" && !pickInteractive {
			return nil
		}
		wt, err := resolveWorktree(cmd.Context())
		if err != nil {
			return err
		}
		cfg.UseWorktree(wt.Path, worktree.SafeName(wt.Name), cmd.Flags().Changed("stack"))
		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err == nil {
		return
	}
	var exitErr *docker.ExitError
	if errors.As(err, &exitErr) {
		os.Exit(exitErr.Code)
	}
	fmt.Println(err)
	os.Exit(1)
}

func resolveWorktree(ctx context.Context) (worktree.Worktree, error) {
	if pickInteractive {
		return pickWorktree(ctx)
	}
	return worktree.Find(ctx, cfg.AspenCloneDir, worktreeName)
}

func init() {
	var err error
	cfg, err = config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}

	rootCmd.PersistentFlags().StringVar(&cfg.StackName, "stack", cfg.StackName, "Docker compose project (stack) name")
	rootCmd.PersistentFlags().StringVarP(&worktreeName, "worktree", "w", "", "Target a git worktree of $ASPEN_CLONE by directory or branch name")
	rootCmd.PersistentFlags().BoolVarP(&pickInteractive, "interactive", "W", false, "Pick the target worktree with a fuzzy finder (-w seeds the query)")
}

func pickWorktree(ctx context.Context) (worktree.Worktree, error) {
	trees, err := worktree.List(ctx, cfg.AspenCloneDir)
	if err != nil {
		return worktree.Worktree{}, err
	}
	idx, err := fuzzyfinder.Find(
		trees,
		func(i int) string {
			if trees[i].Branch == "" {
				return trees[i].Name
			}
			return trees[i].Name + " (" + trees[i].Branch + ")"
		},
		fuzzyfinder.WithQuery(worktreeName),
	)
	if err != nil {
		return worktree.Worktree{}, fmt.Errorf("selection cancelled")
	}
	return trees[idx], nil
}

func resolveContainerConfig(runner *docker.SDKRunner) {
	env, err := runner.ContainerEnv(context.Background(), cfg.MainContainerName())
	if err != nil {
		return
	}
	cfg.ApplyContainerEnv(env)
}
