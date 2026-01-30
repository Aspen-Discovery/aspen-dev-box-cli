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
	rootCmd.AddCommand(CSSCommand())
}

func CSSCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "compilecss",
		Short: "Compile CSS files",
		Long: `Compile CSS files using LESS.
This command compiles the main.less file into main.css in the specified directory.
Use the --rtl flag to compile RTL (right-to-left) CSS files.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			rtl, _ := cmd.Flags().GetBool("rtl")
			cssDir := config.GetCSSDir(rtl)

			if _, err := os.Stat(cssDir); os.IsNotExist(err) {
				return fmt.Errorf("CSS directory does not exist: %s", cssDir)
			}

			runner, err := docker.NewRunner()
			if err != nil {
				return fmt.Errorf("initialize docker: %w", err)
			}
			defer runner.Close()

			result, err := runner.Run(context.Background(), docker.RunConfig{
				Image:      config.GetLessImage(),
				Cmd:        []string{config.GetLessInputFile(), config.GetLessOutputFile()},
				WorkingDir: "/src",
				Binds:      []string{fmt.Sprintf("%s:/src", cssDir)},
			})
			if err != nil {
				return fmt.Errorf("compile CSS: %w", err)
			}

			fmt.Print(result.Stdout)
			if result.Stderr != "" {
				fmt.Fprint(os.Stderr, result.Stderr)
			}

			if result.ExitCode != 0 {
				return fmt.Errorf("compilation failed with exit code %d", result.ExitCode)
			}

			fmt.Printf("Successfully compiled CSS in %s\n", cssDir)
			return nil
		},
	}

	cmd.Flags().BoolP("rtl", "r", false, "Compile RTL (right-to-left) CSS files")
	return cmd
}
