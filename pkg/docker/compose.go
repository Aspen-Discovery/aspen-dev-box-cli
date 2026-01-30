package docker

import (
	"fmt"
	"os"
	"os/exec"
)

type ComposeConfig struct {
	Files    []string
	Detached bool
}

type Compose struct {
	config ComposeConfig
}

func NewCompose(cfg ComposeConfig) *Compose {
	return &Compose{config: cfg}
}

func (c *Compose) Up() error {
	args := c.buildBaseArgs()
	args = append(args, "up")

	if c.config.Detached {
		args = append(args, "-d")
	}

	return c.run(args)
}

func (c *Compose) Down() error {
	args := c.buildBaseArgs()
	args = append(args, "down", "--remove-orphans")
	return c.run(args)
}

func (c *Compose) buildBaseArgs() []string {
	args := []string{"compose"}
	for _, f := range c.config.Files {
		args = append(args, "-f", f)
	}
	return args
}

func (c *Compose) run(args []string) error {
	cmd := exec.Command("docker", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker compose: %w", err)
	}
	return nil
}
