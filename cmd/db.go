package cmd

import (
	"compress/gzip"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"adb/pkg/docker"

	"github.com/spf13/cobra"
)

const defaultSnapshotName = "latest"

func init() {
	rootCmd.AddCommand(DBCommand())
}

func DBCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "db",
		Short: "Open the database shell, or dump and restore snapshots",
		Long: `Without a subcommand, open an interactive MariaDB shell connected to the
Aspen database.

Snapshots let you keep a database across adb down: dump before, restore
after. They are stored under $ASPEN_DOCKER/.cache/snapshots per stack.

Examples:
  adb db
  adb db dump                      # .cache/snapshots/<stack>-latest.sql.gz
  adb db dump before-migration
  adb db restore before-migration
  adb db snapshots`,
		RunE: func(cmd *cobra.Command, args []string) error {
			runner, err := docker.NewRunner()
			if err != nil {
				return fmt.Errorf("initialize docker: %w", err)
			}
			defer runner.Close()
			resolveContainerConfig(runner)

			return runner.ExecInteractive(cmd.Context(), docker.ExecConfig{
				Container: cfg.DBContainerName(),
				Cmd:       []string{"/bin/bash", "-c", "mariadb " + cfg.DBConnectionString()},
			})
		},
	}
	cmd.AddCommand(dbDumpCommand(), dbRestoreCommand(), dbSnapshotsCommand())
	return cmd
}

func dbDumpCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "dump [name]",
		Short: "Dump the Aspen database to a gzipped snapshot",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			runner, err := dbRunner(cmd.Context())
			if err != nil {
				return err
			}
			defer runner.Close()

			path := cfg.SnapshotPath(cfg.StackName, snapshotName(args))
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				return err
			}
			if err := dumpDatabase(cmd.Context(), runner, path); err != nil {
				os.Remove(path)
				return err
			}
			info, err := os.Stat(path)
			if err != nil {
				return err
			}
			fmt.Printf("Dumped %s to %s (%s)\n", cfg.DBName, path, humanSize(info.Size()))
			return nil
		},
	}
}

func dbRestoreCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "restore [name]",
		Short: "Restore the Aspen database from a snapshot",
		Long: `Replace the contents of the Aspen database with a snapshot taken by adb db
dump. Tables in the snapshot are dropped and recreated; tables the snapshot
does not know about are left alone. Run adb updatedb afterwards if your
checkout has moved on since the snapshot.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := cfg.SnapshotPath(cfg.StackName, snapshotName(args))
			file, err := os.Open(path)
			if err != nil {
				return fmt.Errorf("no snapshot at %s (see adb db snapshots)", path)
			}
			defer file.Close()

			runner, err := dbRunner(cmd.Context())
			if err != nil {
				return err
			}
			defer runner.Close()

			if err := restoreDatabase(cmd.Context(), runner, file); err != nil {
				return err
			}
			fmt.Printf("Restored %s from %s\n", cfg.DBName, path)
			return nil
		},
	}
}

func dbSnapshotsCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "snapshots",
		Short: "List this stack's database snapshots",
		RunE: func(cmd *cobra.Command, args []string) error {
			matches, err := filepath.Glob(cfg.SnapshotPath(cfg.StackName, "*"))
			if err != nil {
				return err
			}
			if len(matches) == 0 {
				fmt.Printf("No snapshots for %s yet. Take one with: adb db dump\n", cfg.StackName)
				return nil
			}
			w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			fmt.Fprintln(w, "NAME\tSIZE\tTAKEN")
			for _, path := range matches {
				info, err := os.Stat(path)
				if err != nil {
					continue
				}
				fmt.Fprintf(w, "%s\t%s\t%s\n", snapshotNameFromPath(path), humanSize(info.Size()), info.ModTime().Format("2006-01-02 15:04"))
			}
			return w.Flush()
		},
	}
}

func dbRunner(ctx context.Context) (*docker.SDKRunner, error) {
	runner, err := docker.NewRunner()
	if err != nil {
		return nil, fmt.Errorf("initialize docker: %w", err)
	}
	running, err := runner.ContainerRunning(ctx, cfg.DBContainerName())
	if err != nil {
		runner.Close()
		return nil, err
	}
	if !running {
		runner.Close()
		return nil, fmt.Errorf("%s is not running, start the stack first with 'adb up -d'", cfg.DBContainerName())
	}
	resolveContainerConfig(runner)
	return runner, nil
}

func dumpDatabase(ctx context.Context, runner *docker.SDKRunner, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	gz := gzip.NewWriter(file)

	dumpCmd := "mariadb-dump --single-transaction --quick --routines --triggers --add-drop-table " + cfg.DBConnectionString()
	exitCode, err := runner.ExecPipe(ctx, docker.ExecConfig{
		Container: cfg.DBContainerName(),
		Cmd:       []string{"/bin/bash", "-c", dumpCmd},
	}, nil, gz, os.Stderr)
	if err != nil {
		return err
	}
	if err := gz.Close(); err != nil {
		return err
	}
	if exitCode != 0 {
		return fmt.Errorf("mariadb-dump exited with code %d", exitCode)
	}
	return nil
}

func restoreDatabase(ctx context.Context, runner *docker.SDKRunner, file *os.File) error {
	gz, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("%s is not a gzipped snapshot: %w", file.Name(), err)
	}
	defer gz.Close()

	exitCode, err := runner.ExecPipe(ctx, docker.ExecConfig{
		Container: cfg.DBContainerName(),
		Cmd:       []string{"/bin/bash", "-c", "mariadb " + cfg.DBConnectionString()},
	}, gz, os.Stdout, os.Stderr)
	if err != nil {
		return err
	}
	if exitCode != 0 {
		return fmt.Errorf("mariadb exited with code %d", exitCode)
	}
	return nil
}

func snapshotName(args []string) string {
	if len(args) == 0 {
		return defaultSnapshotName
	}
	return args[0]
}

func snapshotNameFromPath(path string) string {
	base := strings.TrimSuffix(filepath.Base(path), ".sql.gz")
	return strings.TrimPrefix(base, cfg.StackName+"-")
}

func humanSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	value, suffix := float64(bytes)/unit, "KB"
	if value >= unit {
		value, suffix = value/unit, "MB"
	}
	if value >= unit {
		value, suffix = value/unit, "GB"
	}
	return fmt.Sprintf("%.1f %s", value, suffix)
}
