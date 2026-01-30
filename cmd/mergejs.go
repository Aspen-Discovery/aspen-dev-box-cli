package cmd

import (
	"context"
	"fmt"
	"os"

	"adb/pkg/config"
	"adb/pkg/docker"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(MergeJSCommand())
}

func MergeJSCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "mergejs",
		Short: "Merge JavaScript files",
		Long: `Merge JavaScript files using the merge_javascript.php script.
This command runs the merge script inside the main container to combine and minify JavaScript files.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			runner, err := docker.NewRunner()
			if err != nil {
				return fmt.Errorf("initialize docker: %w", err)
			}
			defer runner.Close()

			result, err := runner.Exec(context.Background(), docker.ExecConfig{
				Container:  config.GetMainContainerName(),
				Cmd:        []string{"php", config.GetMergeJSScript()},
				WorkingDir: config.GetJSWorkDir(),
			})
			if err != nil {
				return fmt.Errorf("merge javascript: %w", err)
			}

			fmt.Print(result.Stdout)
			if result.Stderr != "" {
				fmt.Fprint(os.Stderr, result.Stderr)
			}

			if result.ExitCode != 0 {
				return fmt.Errorf("merge failed with exit code %d", result.ExitCode)
			}

			return nil
		},
	}
}
