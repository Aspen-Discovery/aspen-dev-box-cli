package cmd

import (
	"context"
	"fmt"

	"adb/pkg/docker"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(OAuthCommand())
}

func OAuthCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "oauth <client_id> <client_secret>",
		Short: "Update OAuth credentials",
		Long: `Update the OAuth client ID and secret for ILS logins.
This command updates the OAuth credentials in the database for the specified driver.
By default, it updates the Koha driver credentials.`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientID := args[0]
			clientSecret := args[1]

			printRows, _ := cmd.Flags().GetBool("print")
			driver, _ := cmd.Flags().GetString("driver")
			if driver == "" {
				driver = "Koha"
			}

			sql := fmt.Sprintf(`
SET @update_count = 0;
UPDATE account_profiles 
SET oAuthClientId='%s', 
oAuthClientSecret='%s' 
WHERE driver='%s';
SET @update_count = ROW_COUNT();
SELECT @update_count as Changed_Rows;
`, clientID, clientSecret, driver)

			if printRows {
				sql += fmt.Sprintf(`
SELECT * FROM account_profiles 
WHERE driver='%s'\G
`, driver)
			}

			runner, err := docker.NewRunner()
			if err != nil {
				return fmt.Errorf("initialize docker: %w", err)
			}
			defer runner.Close()
			resolveContainerConfig(runner)

			shellCmd := fmt.Sprintf("echo \"%s\" | mariadb %s", sql, cfg.DBConnectionString())

			return runner.ExecInteractive(context.Background(), docker.ExecConfig{
				Container: cfg.DBContainerName,
				Cmd:       []string{"/bin/bash", "-c", shellCmd},
			})
		},
	}

	cmd.Flags().StringP("driver", "d", "", "Specify the driver (default is 'Koha')")
	cmd.Flags().BoolP("print", "p", false, "Print the rows that match the driver")

	return cmd
}
