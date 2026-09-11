package cmd

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"adb/pkg/docker"

	"github.com/spf13/cobra"
)

const kohaOAuthService = "koha-oauth-init"

func init() {
	rootCmd.AddCommand(LsCommand())
}

func LsCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "ls",
		Aliases: []string{"list", "ps"},
		Short:   "List aspen stacks and their URLs",
		Long: `List every aspen stack docker knows about (default instance, worktrees and
named stacks) with the state of each service, its URL, the clone it serves
and the ILS it is wired to, including whether the Koha OAuth setup succeeded.
The stack adb currently targets is marked with *.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			runner, err := docker.NewRunner()
			if err != nil {
				return fmt.Errorf("initialize docker: %w", err)
			}
			defer runner.Close()

			stacks, err := runner.StackSummaries(cmd.Context(), cfg.MainContainerService)
			if err != nil {
				return err
			}
			if len(stacks) == 0 {
				fmt.Println("No aspen stacks found. Start one with: adb up -d")
				return nil
			}
			printStacks(stacks)
			return nil
		},
	}
}

func printStacks(stacks []docker.StackSummary) {
	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "  STACK\tASPEN\tDB\tSOLR\tURL\tILS\tCLONE")
	for _, s := range stacks {
		marker := " "
		if s.Project == cfg.StackName {
			marker = "*"
		}
		fmt.Fprintf(w, "%s %s\t%s\t%s\t%s\t%s\t%s\t%s\n", marker, s.Project,
			stackStatus(s), s.Services[cfg.DBContainerService], s.Services["solr"], s.URL, ilsStatus(s), s.Clone)
	}
	w.Flush()
}

func stackStatus(s docker.StackSummary) string {
	if s.Health == "" {
		return s.State
	}
	if s.State == "running" {
		return s.Health
	}
	return s.State
}

func ilsStatus(s docker.StackSummary) string {
	if s.ILS != "koha" {
		return s.ILS
	}
	oauth, ok := s.Services[kohaOAuthService]
	if !ok {
		return s.ILS
	}
	if oauth == "exited(0)" {
		return "koha (oauth ok)"
	}
	if strings.HasPrefix(oauth, "exited") {
		return "koha (oauth failed, see adb logs --service " + kohaOAuthService + ")"
	}
	return "koha (oauth pending)"
}
