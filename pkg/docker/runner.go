package docker

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

type RunConfig struct {
	Image      string
	Cmd        []string
	WorkingDir string
	User       string
	Binds      []string
	Env        []string
}

type RunResult struct {
	Stdout   string
	Stderr   string
	ExitCode int64
}

type Runner interface {
	Run(ctx context.Context, cfg RunConfig) (*RunResult, error)
	Close() error
}

type SDKRunner struct {
	client *client.Client
}

func NewRunner() (*SDKRunner, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("create docker client: %w", err)
	}
	return &SDKRunner{client: cli}, nil
}

func (r *SDKRunner) Close() error {
	return r.client.Close()
}

func (r *SDKRunner) Run(ctx context.Context, cfg RunConfig) (*RunResult, error) {
	if err := r.pullImageIfNeeded(ctx, cfg.Image); err != nil {
		return nil, err
	}

	containerID, err := r.createContainer(ctx, cfg)
	if err != nil {
		return nil, err
	}
	defer r.removeContainer(ctx, containerID)

	if err := r.client.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
		return nil, fmt.Errorf("start container: %w", err)
	}

	exitCode, err := r.waitForContainer(ctx, containerID)
	if err != nil {
		return nil, err
	}

	stdout, stderr, err := r.getLogs(ctx, containerID)
	if err != nil {
		return nil, err
	}

	return &RunResult{
		Stdout:   stdout,
		Stderr:   stderr,
		ExitCode: exitCode,
	}, nil
}

func (r *SDKRunner) pullImageIfNeeded(ctx context.Context, imageName string) error {
	_, _, err := r.client.ImageInspectWithRaw(ctx, imageName)
	if err == nil {
		return nil
	}

	reader, err := r.client.ImagePull(ctx, imageName, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("pull image %s: %w", imageName, err)
	}
	defer reader.Close()
	io.Copy(io.Discard, reader)
	return nil
}

func (r *SDKRunner) createContainer(ctx context.Context, cfg RunConfig) (string, error) {
	containerCfg := &container.Config{
		Image:      cfg.Image,
		Cmd:        cfg.Cmd,
		WorkingDir: cfg.WorkingDir,
		User:       cfg.User,
		Tty:        false,
		Env:        cfg.Env,
	}

	hostCfg := &container.HostConfig{
		Binds: cfg.Binds,
	}

	resp, err := r.client.ContainerCreate(ctx, containerCfg, hostCfg, nil, nil, "")
	if err != nil {
		return "", fmt.Errorf("create container: %w", err)
	}
	return resp.ID, nil
}

func (r *SDKRunner) waitForContainer(ctx context.Context, containerID string) (int64, error) {
	statusCh, errCh := r.client.ContainerWait(ctx, containerID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			return -1, fmt.Errorf("wait container: %w", err)
		}
		return -1, nil
	case status := <-statusCh:
		return status.StatusCode, nil
	case <-ctx.Done():
		return -1, ctx.Err()
	}
}

func (r *SDKRunner) getLogs(ctx context.Context, containerID string) (string, string, error) {
	logReader, err := r.client.ContainerLogs(ctx, containerID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
	})
	if err != nil {
		return "", "", fmt.Errorf("get logs: %w", err)
	}
	defer logReader.Close()

	var stdoutBuf, stderrBuf bytes.Buffer
	stdcopy.StdCopy(&stdoutBuf, &stderrBuf, logReader)
	return stdoutBuf.String(), stderrBuf.String(), nil
}

func (r *SDKRunner) removeContainer(ctx context.Context, containerID string) {
	r.client.ContainerRemove(ctx, containerID, container.RemoveOptions{
		Force:         true,
		RemoveVolumes: true,
	})
}
