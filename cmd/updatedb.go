package cmd

import (
	"context"
	"fmt"

	"adb/pkg/config"
	"adb/pkg/docker"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(UpdateDBCommand())
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

			curlCmd := "curl -k http://localhost/API/SystemAPI?method=runPendingDatabaseUpdates"

			err = runner.ExecInteractive(context.Background(), docker.ExecConfig{
				Container: config.GetMainContainerName(),
				Cmd:       []string{"/bin/bash", "-c", curlCmd},
			})
			if err != nil {
				return err
			}

			fmt.Println("\nDatabase updates completed successfully")
			return nil
		},
	}
}
