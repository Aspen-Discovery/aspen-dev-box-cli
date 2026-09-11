package docker

import (
	"context"
	"fmt"
	"os"
	"os/exec"
)

type ComposeConfig struct {
	Project  string
	Files    []string
	Detached bool
	Wait     bool
	Quiet    bool
}

type Composer interface {
	Up(ctx context.Context) error
	Pull(ctx context.Context) error
	Down(ctx context.Context) error
	DownKeepingVolumes(ctx context.Context) error
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
	if c.config.Wait {
		args = append(args, "--wait")
	}

	return c.run(ctx, args)
}

func (c *Compose) Pull(ctx context.Context) error {
	args := c.baseArgs()
	args = append(args, "pull")
	return c.run(ctx, args)
}

func (c *Compose) Down(ctx context.Context) error {
	return c.down(ctx, false)
}

func (c *Compose) DownKeepingVolumes(ctx context.Context) error {
	return c.down(ctx, true)
}

func (c *Compose) down(ctx context.Context, keepVolumes bool) error {
	args := c.baseArgs()
	args = append(args, "down", "--remove-orphans")
	if !keepVolumes {
		args = append(args, "--volumes")
	}
	return c.run(ctx, args)
}

func (c *Compose) Stop(ctx context.Context, services ...string) error {
	return c.run(ctx, append(append(c.baseArgs(), "stop"), services...))
}

func (c *Compose) Start(ctx context.Context, services ...string) error {
	return c.run(ctx, append(append(c.baseArgs(), "start"), services...))
}

func (c *Compose) Restart(ctx context.Context, services ...string) error {
	return c.run(ctx, append(append(c.baseArgs(), "restart"), services...))
}

func (c *Compose) RunService(ctx context.Context, service string, extra []string) error {
	args := c.baseArgs()
	args = append(args, "run", "--rm", "--no-deps", service)
	args = append(args, extra...)
	return c.run(ctx, args)
}

func (c *Compose) baseArgs() []string {
	args := []string{"compose"}
	if c.config.Quiet {
		args = append(args, "--progress", "quiet")
	}
	if c.config.Project != "" {
		args = append(args, "-p", c.config.Project)
	}
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
