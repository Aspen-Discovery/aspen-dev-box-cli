package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"adb/pkg/docker"

	"github.com/fatih/color"
	"github.com/k3a/html2text"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(UpdateDBCommand())
}

type updateDBResponse struct {
	Result struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	} `json:"result"`
}

func UpdateDBCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "updatedb",
		Short: "Run database updates",
		Long: `Run any pending database updates for Aspen Discovery.
This command triggers the database update process by calling the SystemAPI endpoint.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			runner, err := docker.NewRunner()
			if err != nil {
				return fmt.Errorf("initialize docker: %w", err)
			}
			defer runner.Close()

			curlCmd := "curl -s -k http://localhost/API/SystemAPI?method=runPendingDatabaseUpdates"

			result, err := runner.Exec(context.Background(), docker.ExecConfig{
				Container: cfg.MainContainerName,
				Cmd:       []string{"/bin/bash", "-c", curlCmd},
			})
			if err != nil {
				return err
			}

			if result.ExitCode != 0 {
				return fmt.Errorf("curl failed with exit code %d: %s", result.ExitCode, result.Stderr)
			}

			return formatUpdateDBOutput(result.Stdout)
		},
	}
}

var (
	updateNamePattern = regexp.MustCompile(`^[A-Z][a-zA-Z\s]+ - .+$`)
	sqlPattern        = regexp.MustCompile(`^(ALTER|CREATE|UPDATE|INSERT|DROP|SELECT)\s`)

	headerStyle  = color.New(color.Bold)
	successStyle = color.New(color.FgGreen, color.Bold)
	warningStyle = color.New(color.FgYellow, color.Bold)
	errorStyle   = color.New(color.FgRed)
	nameStyle    = color.New(color.FgBlue, color.Bold)
	sqlStyle     = color.New(color.FgCyan)
)

func formatUpdateDBOutput(output string) error {
	var resp updateDBResponse
	if err := json.Unmarshal([]byte(output), &resp); err != nil {
		fmt.Println(output)
		return nil
	}

	if resp.Result.Success {
		successStyle.Println("✓ Database updates completed successfully")
	} else {
		warningStyle.Println("⚠ Database updates completed with issues")
	}
	fmt.Println()

	text := html2text.HTML2Text(resp.Result.Message)

	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)

		switch {
		case line == "":
			fmt.Println()
		case updateNamePattern.MatchString(line):
			fmt.Println()
			nameStyle.Println(line)
		case sqlPattern.MatchString(line):
			sqlStyle.Printf("  %s\n", line)
		case strings.HasPrefix(line, "Update failed:") || strings.HasPrefix(line, "Stack trace:") || strings.HasPrefix(line, "#"):
			errorStyle.Printf("  %s\n", line)
		default:
			fmt.Printf("  %s\n", line)
		}
	}

	return nil
}
