package docker

import (
	"context"
	"fmt"
	"os"
	"os/exec"
)

type ComposeConfig struct {
	Files    []string
	Detached bool
}

type Composer interface {
	Up(ctx context.Context) error
	Pull(ctx context.Context) error
	Down(ctx context.Context) error
}

type Compose struct {
	config ComposeConfig
}

func NewCompose(cfg ComposeConfig) *Compose {
	return &Compose{config: cfg}
}

func (c *Compose) Up(ctx context.Context) error {
	args := c.baseArgs()
	args = append(args, "up")

	if c.config.Detached {
		args = append(args, "-d")
	}

	return c.run(ctx, args)
}

func (c *Compose) Pull(ctx context.Context) error {
	args := c.baseArgs()
	args = append(args, "pull")
	return c.run(ctx, args)
}

func (c *Compose) Down(ctx context.Context) error {
	args := c.baseArgs()
	args = append(args, "down", "--remove-orphans")
	return c.run(ctx, args)
}

func (c *Compose) baseArgs() []string {
	args := []string{"compose"}
	for _, f := range c.config.Files {
		args = append(args, "-f", f)
	}
	return args
}

func (c *Compose) run(ctx context.Context, args []string) error {
	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return fmt.Errorf("docker compose exited with code %d", exitErr.ExitCode())
		}
		return fmt.Errorf("docker compose: %w", err)
	}
	return nil
}
