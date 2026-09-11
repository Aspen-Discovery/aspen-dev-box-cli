package cmd

import (
	"context"
	"fmt"
	"strconv"

	"adb/pkg/docker"
	"adb/pkg/jar"

	"github.com/spf13/cobra"
)

const jdwpContainerPort = "5005"

func init() {
	rootCmd.AddCommand(DebugCommand())
}

func DebugCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "debug <module> [extra args...]",
		Short: "Run a java module under JDWP so an IDE can attach",
		Long: `Compile a java module from source inside the main container and run it with
the JDWP agent, suspended until a debugger attaches. The stack must be up with
the java debug overlay (adb up -j) so port 5005 reaches the host.

Modules are discovered from the aspen clone: any directory under code/ with a
META-INF/MANIFEST.MF. The main class comes from that manifest and the shared
java libraries are compiled in when the module imports them. Extra arguments
are passed to the job after the site name.

Examples:
  adb debug list
  adb debug reindexer
  adb debug koha_export`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if args[0] == "list" {
				return listDebuggableModules()
			}
			module, err := jar.FindModule(cfg.CodeDir(), args[0])
			if err != nil {
				return err
			}
			mainClass, err := module.MainClass()
			if err != nil {
				return err
			}

			runner, err := docker.NewRunner()
			if err != nil {
				return fmt.Errorf("initialize docker: %w", err)
			}
			defer runner.Close()

			hostPort, err := debugHostPort(cmd.Context(), runner)
			if err != nil {
				return err
			}
			fmt.Printf("Attach your debugger to localhost:%s once compilation finishes\n", hostPort)
			return runner.ExecInteractive(cmd.Context(), docker.ExecConfig{
				Container: cfg.MainContainerName(),
				User:      "www-data",
				Env: []string{
					"MODULE=" + module.Name,
					"MAIN_CLASS=" + mainClass,
					"NEEDS_SHARED_LIBS=" + strconv.FormatBool(module.NeedsSharedLib),
					"DEBUG_PORT=" + jdwpContainerPort,
				},
				Cmd: append([]string{"bash", "-c", jar.DebugScript, "adb-debug"}, args[1:]...),
			})
		},
	}
}

func listDebuggableModules() error {
	modules, err := jar.DiscoverModules(cfg.CodeDir(), cfg.ExcludedJarPatterns)
	if err != nil {
		return err
	}
	for _, m := range modules {
		fmt.Println(m.Name)
	}
	return nil
}

func debugHostPort(ctx context.Context, runner *docker.SDKRunner) (string, error) {
	running, err := runner.ContainerRunning(ctx, cfg.MainContainerName())
	if err != nil {
		return "", err
	}
	if !running {
		return "", fmt.Errorf("%s is not running, start the stack with 'adb up -d -j'", cfg.MainContainerName())
	}
	hostPort, err := runner.PublishedHostPort(ctx, cfg.MainContainerName(), jdwpContainerPort)
	if err != nil {
		return "", err
	}
	if hostPort == "" {
		return "", fmt.Errorf("port %s is not published on %s, restart the stack with 'adb up -d -j'", jdwpContainerPort, cfg.MainContainerName())
	}
	return hostPort, nil
}
