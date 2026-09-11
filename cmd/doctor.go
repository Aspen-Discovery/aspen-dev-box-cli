package cmd

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"adb/pkg/docker"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

const skipConfigValidation = "skipConfigValidation"

var (
	passStyle = color.New(color.FgGreen, color.Bold)
	warnStyle = color.New(color.FgYellow, color.Bold)
	failStyle = color.New(color.FgRed, color.Bold)
)

type checkup struct {
	failed bool
}

func (c *checkup) pass(format string, args ...any) {
	passStyle.Print("✓ ")
	fmt.Printf(format+"\n", args...)
}

func (c *checkup) warn(format string, args ...any) {
	warnStyle.Print("! ")
	fmt.Printf(format+"\n", args...)
}

func (c *checkup) fail(format string, args ...any) {
	c.failed = true
	failStyle.Print("✗ ")
	fmt.Printf(format+"\n", args...)
}

func init() {
	rootCmd.AddCommand(DoctorCommand())
}

func DoctorCommand() *cobra.Command {
	var kohaStack string
	cmd := &cobra.Command{
		Use:         "doctor",
		Short:       "Check the machine is set up for the aspen dev box",
		Annotations: map[string]string{skipConfigValidation: "true"},
		Long: `Check everything adb up needs before it runs: the ASPEN_DOCKER and
ASPEN_CLONE environment variables, the .env file, the host UID/GID mapping,
the docker daemon and compose, the koha-testing-docker network the default
--ils koha needs, and the host ports the proxy binds.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c := &checkup{}
			checkEnvironment(c)
			checkDocker(cmd.Context(), c, kohaStack)
			if c.failed {
				return fmt.Errorf("some checks failed")
			}
			fmt.Println("\nAll good. Start the stack with: adb up -d")
			return nil
		},
	}
	cmd.Flags().StringVarP(&kohaStack, "koha-stack", "k", "kohadev", "koha-testing-docker stack adb up would connect to")
	return cmd
}

func checkEnvironment(c *checkup) {
	if cfg.ProjectsDir == "" {
		c.fail("ASPEN_DOCKER is not set (path to the aspen-dev-box-image clone)")
	} else if _, err := os.Stat(cfg.ComposeFilePath("docker-compose.yml")); err != nil {
		c.fail("ASPEN_DOCKER=%s has no compose/docker-compose.yml", cfg.ProjectsDir)
	} else {
		c.pass("ASPEN_DOCKER=%s", cfg.ProjectsDir)
	}

	if cfg.AspenCloneDir == "" {
		c.fail("ASPEN_CLONE is not set (path to your aspen-discovery clone)")
	} else if _, err := os.Stat(filepath.Join(cfg.AspenCloneDir, "install", "aspen.sql")); err != nil {
		c.fail("ASPEN_CLONE=%s has no install/aspen.sql, is it an aspen-discovery checkout?", cfg.AspenCloneDir)
	} else {
		c.pass("ASPEN_CLONE=%s", cfg.AspenCloneDir)
	}

	if cfg.ProjectsDir != "" {
		checkEnvFile(c)
	}

	uid, gid := os.Getenv("UID"), os.Getenv("GID")
	if uid == "" || gid == "" {
		c.warn("UID/GID could not be determined, containers will fall back to 501/20")
		return
	}
	c.pass("container users will map to UID=%s GID=%s", uid, gid)
	if uid != strconv.Itoa(os.Getuid()) && os.Getuid() >= 0 {
		c.warn("UID=%s differs from your user id %d, bind-mounted files may end up owned by someone else", uid, os.Getuid())
	}
}

func checkEnvFile(c *checkup) {
	envPath := cfg.EnvFilePath()
	if _, err := os.Stat(envPath); err != nil {
		c.fail(".env missing at %s (copy .env.example to .env)", envPath)
		return
	}
	c.pass(".env found at %s", envPath)
}

func checkDocker(ctx context.Context, c *checkup, kohaStack string) {
	runner, err := docker.NewRunner()
	if err != nil {
		c.fail("docker client: %v", err)
		return
	}
	defer runner.Close()

	if err := runner.Ping(ctx); err != nil {
		c.fail("docker daemon not reachable: %v (is Docker running?)", err)
		return
	}
	c.pass("docker daemon reachable")

	if out, err := exec.CommandContext(ctx, "docker", "compose", "version", "--short").Output(); err != nil {
		c.fail("docker compose v2 not available: %v", err)
	} else {
		c.pass("docker compose %s", trimNewline(string(out)))
	}

	checkKohaNetwork(ctx, c, runner, kohaStack)
	checkProxyPorts(ctx, c, runner)
}

func checkKohaNetwork(ctx context.Context, c *checkup, runner *docker.SDKRunner, kohaStack string) {
	network := kohaStack + "_kohanet"
	exists, err := runner.NetworkExists(ctx, network)
	if err != nil {
		c.fail("inspect network %s: %v", network, err)
		return
	}
	if exists {
		c.pass("koha-testing-docker network %s present (default --ils koha will work)", network)
		return
	}
	c.warn("network %s not found: start koha-testing-docker first, or use adb up --ils none", network)
}

func checkProxyPorts(ctx context.Context, c *checkup, runner *docker.SDKRunner) {
	running, port, err := runner.ProxyInfo(ctx, proxyNetwork)
	if err != nil {
		c.fail("inspect aspen proxy: %v", err)
		return
	}
	if running {
		c.pass("aspen proxy running on port %d", port)
		return
	}
	for _, p := range proxyPorts() {
		checkPort(ctx, c, runner, p)
	}
}

func proxyPorts() []string {
	return []string{
		envOr("PROXY_HTTP_PORT", "8083"),
		envOr("PROXY_SOLR_PORT", "8084"),
		envOr("PROXY_DBGUI_PORT", "8085"),
		envOr("PROXY_DASHBOARD_PORT", "8090"),
	}
}

func checkPort(ctx context.Context, c *checkup, runner *docker.SDKRunner, port string) {
	listener, err := net.Listen("tcp", "127.0.0.1:"+port)
	if err == nil {
		listener.Close()
		c.pass("port %s free for the aspen proxy", port)
		return
	}
	owner, lookupErr := runner.PublishedPortOwner(ctx, port)
	if lookupErr == nil && owner != "" {
		c.warn("port %s is published by container %s, the proxy will not be able to bind it", port, owner)
		return
	}
	c.fail("port %s is in use by another process (set PROXY_*_PORT in .env or free it)", port)
}

func envOr(name, fallback string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return fallback
}

func trimNewline(s string) string {
	for len(s) > 0 && (s[len(s)-1] == '\n' || s[len(s)-1] == '\r') {
		s = s[:len(s)-1]
	}
	return s
}
