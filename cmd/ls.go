package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"adb/pkg/docker"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(LsCommand())
}

func LsCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "ls",
		Aliases: []string{"list", "ps"},
		Short:   "List aspen stacks and their URLs",
		Long: `List every aspen stack docker knows about (default instance, worktrees and
named stacks), with its container state, URL, the clone it serves and the ILS
it is wired to. The stack adb currently targets is marked with *.`,
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
	fmt.Fprintln(w, "  STACK\tSTATUS\tURL\tILS\tCLONE")
	for _, s := range stacks {
		marker := " "
		if s.Project == cfg.StackName {
			marker = "*"
		}
		fmt.Fprintf(w, "%s %s\t%s\t%s\t%s\t%s\n", marker, s.Project, stackStatus(s), s.URL, s.ILS, s.Clone)
	}
	w.Flush()
}

func stackStatus(s docker.StackSummary) string {
	if s.Health == "" {
		return s.State
	}
	return s.State + " (" + s.Health + ")"
}
