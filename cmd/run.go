package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"adb/pkg/docker"

	"github.com/spf13/cobra"
)

type aspenJob struct {
	Dir string
	Bin []string
}

var phpJobs = map[string]aspenJob{
	"cron":     {Dir: "/usr/local/aspen-discovery/docker/files/cron", Bin: []string{"php", "checkBackgroundProcessesDocker.php"}},
	"sitemaps": {Dir: "/usr/local/aspen-discovery/code/web/cron", Bin: []string{"php", "createSitemaps.php"}},
}

func init() {
	rootCmd.AddCommand(RunCommand())
}

func RunCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "run <job> [extra args...]",
		Short: "Run an aspen background job (reindexer, koha_export, etc.)",
		Long:  "Run 'adb run list' to see available jobs.\n\nJAR jobs are discovered from the aspen clone; any module under code/ with a\nbuilt <module>.jar is runnable by its module name.\n\nExamples:\n  adb run koha_export\n  adb run reindexer nightly\n  adb run list",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			jobs, err := discoverJobs(cfg.CodeDir())
			if err != nil {
				return err
			}
			if args[0] == "list" {
				for _, name := range sortedJobs(jobs) {
					fmt.Println(name)
				}
				return nil
			}
			runner, err := docker.NewRunner()
			if err != nil {
				return fmt.Errorf("initialize docker: %w", err)
			}
			defer runner.Close()
			return runJob(cmd.Context(), runner, jobs, args[0], args[1:])
		},
	}
}

func discoverJobs(codeDir string) (map[string]aspenJob, error) {
	jobs := make(map[string]aspenJob, len(phpJobs))
	for name, job := range phpJobs {
		jobs[name] = job
	}

	entries, err := os.ReadDir(codeDir)
	if err != nil {
		return nil, fmt.Errorf("read aspen code dir: %w", err)
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if _, err := os.Stat(filepath.Join(codeDir, name, name+".jar")); err != nil {
			continue
		}
		jobs[name] = aspenJob{
			Dir: "/usr/local/aspen-discovery/code/" + name,
			Bin: []string{"java", "-jar", name + ".jar"},
		}
	}
	return jobs, nil
}

func sortedJobs(jobs map[string]aspenJob) []string {
	names := make([]string, 0, len(jobs))
	for k := range jobs {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}

func runJob(ctx context.Context, runner *docker.SDKRunner, jobs map[string]aspenJob, name string, extra []string) error {
	job, ok := jobs[name]
	if !ok {
		return fmt.Errorf("unknown job %q (try one of: %s)", name, strings.Join(sortedJobs(jobs), ", "))
	}
	parts := append([]string{}, job.Bin...)
	parts = append(parts, "${SITE_NAME:-dev.localhost}")
	parts = append(parts, extra...)
	script := fmt.Sprintf("cd %s && %s", job.Dir, strings.Join(parts, " "))
	return runner.ExecInteractive(ctx, docker.ExecConfig{
		Container: cfg.MainContainerName(),
		User:      "www-data",
		Cmd:       []string{"sh", "-c", script},
	})
}
