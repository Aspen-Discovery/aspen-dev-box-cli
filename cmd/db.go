package cmd

import (
	"context"
	"fmt"

	"adb/pkg/docker"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(DBCommand())
}

func DBCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "db",
		Short: "Opens the database shell",
		Long: `Opens an interactive MariaDB shell connected to the Aspen database.
This command provides direct access to the database for running SQL queries and managing data.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			runner, err := docker.NewRunner()
			if err != nil {
				return fmt.Errorf("initialize docker: %w", err)
			}
			defer runner.Close()

			return runner.ExecInteractive(context.Background(), docker.ExecConfig{
				Container: cfg.DBContainerName,
				Cmd:       []string{"/bin/bash", "-c", "mariadb " + cfg.DBConnectionString()},
			})
		},
	}
}
