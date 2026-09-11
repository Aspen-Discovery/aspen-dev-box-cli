package cmd

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"adb/pkg/docker"

	"github.com/spf13/cobra"
)

const xdebugScript = `set -e
PHP_VERSION=$(php -r 'echo PHP_MAJOR_VERSION, ".", PHP_MINOR_VERSION;')
CONF_DIR="/etc/php/${PHP_VERSION}/fpm/conf.d"
SETTINGS="${CONF_DIR}/99-xdebug.ini"
case "$1" in
on)
    if ! php -r 'exit(file_exists(ini_get("extension_dir") . "/xdebug.so") ? 0 : 1);'; then
        echo "xdebug extension not present in this image (pull an image built with php-xdebug)"
        exit 1
    fi
    rm -f "${SETTINGS}"
    cat > "${SETTINGS}"
    phpenmod xdebug
    pkill -USR2 -o php-fpm
    echo "Xdebug enabled for web requests (PHP-FPM reloaded)"
    ;;
off)
    phpdismod xdebug
    rm -f "${SETTINGS}"
    pkill -USR2 -o php-fpm
    echo "Xdebug disabled (PHP-FPM reloaded)"
    ;;
status)
    MASTER=$(pgrep -o php-fpm || true)
    if [ -n "${MASTER}" ] && grep -q xdebug.so "/proc/${MASTER}/maps" 2>/dev/null; then
        MODE=$("php-fpm${PHP_VERSION}" -i 2>/dev/null | sed -n 's/^xdebug.mode => \([^ ]*\).*/\1/p' | head -n1)
        echo "on (xdebug.mode=${MODE:-default})"
    else
        echo "off"
    fi
    ;;
esac
`

func init() {
	rootCmd.AddCommand(XdebugCommand())
}

func XdebugCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "xdebug on|off|status",
		Short: "Toggle Xdebug for web requests without restarting the stack",
		Long: `Enable or disable the Xdebug extension for PHP-FPM inside the running aspen
container and reload PHP-FPM, so step debugging can be switched on mid-session
instead of restarting with adb up -g. "on" installs the settings from
$ASPEN_DOCKER/xdebug.ini (client host, mode, start_with_request).

Examples:
  adb xdebug on
  adb xdebug status
  adb xdebug off`,
		Args:      cobra.ExactArgs(1),
		ValidArgs: []string{"on", "off", "status"},
		RunE: func(cmd *cobra.Command, args []string) error {
			action := args[0]
			validAction := action == "on" || action == "off" || action == "status"
			if !validAction {
				return fmt.Errorf("expected on, off or status, got %q", action)
			}
			settings, err := xdebugSettings(action)
			if err != nil {
				return err
			}

			runner, err := docker.NewRunner()
			if err != nil {
				return fmt.Errorf("initialize docker: %w", err)
			}
			defer runner.Close()
			return runContainerScript(cmd.Context(), runner, xdebugScript, action, settings)
		},
	}
}

func xdebugSettings(action string) ([]byte, error) {
	if action != "on" {
		return nil, nil
	}
	path := filepath.Join(cfg.ProjectsDir, "xdebug.ini")
	settings, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read xdebug settings: %w", err)
	}
	return settings, nil
}

func runContainerScript(ctx context.Context, runner *docker.SDKRunner, script, arg string, stdin []byte) error {
	var input io.Reader
	if stdin != nil {
		input = bytes.NewReader(stdin)
	}
	exitCode, err := runner.ExecPipe(ctx, docker.ExecConfig{
		Container: cfg.MainContainerName(),
		Cmd:       []string{"bash", "-c", script, "adb", arg},
	}, input, os.Stdout, os.Stderr)
	if err != nil {
		return err
	}
	if exitCode != 0 {
		return &docker.ExitError{Code: exitCode}
	}
	return nil
}
