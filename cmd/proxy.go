package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"adb/pkg/docker"

	"github.com/spf13/cobra"
)

const proxyNetwork = "aspen-proxy"
const proxyStack = "aspen-proxy"

func init() {
	rootCmd.AddCommand(ProxyCommand())
}

func ProxyCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "proxy",
		Short: "Manage the aspen reverse proxy",
		Long: `Manage the aspen traefik proxy that routes every proxied aspen stack by
hostname on the external "aspen-proxy" docker network. It is fully
independent of koha-testing-docker's proxy; if that one holds ports 80/443,
start this one elsewhere with PROXY_HTTP_PORT / PROXY_HTTPS_PORT.

Dashboard: http://localhost:8090/dashboard/ (override the port with
PROXY_DASHBOARD_PORT)`,
	}
	cmd.AddCommand(proxyUpCommand(), proxyDownCommand())
	return cmd
}

func proxyCompose() *docker.Compose {
	return docker.NewCompose(docker.ComposeConfig{
		Project:  proxyStack,
		Files:    []string{filepath.Join(cfg.ProjectsDir, "proxy", "docker-compose.yml")},
		Detached: true,
	})
}

func ensureProxy(ctx context.Context) (uint16, error) {
	runner, err := docker.NewRunner()
	if err != nil {
		return 0, err
	}
	defer runner.Close()

	running, port, err := runner.ProxyInfo(ctx, proxyNetwork)
	if err != nil || running {
		return port, err
	}

	if err := runner.EnsureNetwork(ctx, proxyNetwork); err != nil {
		return 0, err
	}
	if err := proxyCompose().Up(ctx); err != nil {
		return 0, fmt.Errorf("start aspen proxy: %w (if its ports are taken, set PROXY_HTTP_PORT / PROXY_HTTPS_PORT)", err)
	}

	running, port, err = runner.ProxyInfo(ctx, proxyNetwork)
	if err != nil {
		return 0, err
	}
	if !running {
		return 0, fmt.Errorf("aspen proxy started but not detected on the %s network", proxyNetwork)
	}
	fmt.Println("Started the aspen proxy")
	return port, nil
}

func dashboardPort() string {
	if v := os.Getenv("PROXY_DASHBOARD_PORT"); v != "" {
		return v
	}
	return "8090"
}

func maybeDownProxy(ctx context.Context) error {
	runner, err := docker.NewRunner()
	if err != nil {
		return err
	}
	defer runner.Close()

	running, _, err := runner.ProxyInfo(ctx, proxyNetwork)
	if err != nil || !running {
		return err
	}
	stacks, err := runner.ProxiedStacks(ctx, proxyStack)
	if err != nil || stacks > 0 {
		return err
	}
	fmt.Println("No proxied stacks left — stopping the aspen proxy")
	return proxyCompose().Down(ctx)
}

func proxyUpCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "up",
		Short: "Start the aspen proxy (creates the aspen-proxy network if needed)",
		RunE: func(cmd *cobra.Command, args []string) error {
			port, err := ensureProxy(cmd.Context())
			if err != nil {
				return err
			}
			fmt.Printf("The aspen proxy is up on port %d\n", port)
			fmt.Printf("Dashboard: http://localhost:%s/dashboard/\n", dashboardPort())
			return nil
		},
	}
}

func proxyDownCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "down",
		Short: "Stop the aspen proxy",
		RunE: func(cmd *cobra.Command, args []string) error {
			return proxyCompose().Down(cmd.Context())
		},
	}
}
