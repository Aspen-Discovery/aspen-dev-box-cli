package docker

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/go-connections/nat"
	"github.com/moby/term"
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

type ExecConfig struct {
	Container  string
	Cmd        []string
	WorkingDir string
	User       string
	Env        []string
}

type ExitError struct {
	Code int
}

func (e *ExitError) Error() string {
	return fmt.Sprintf("exited with code %d", e.Code)
}

type Runner interface {
	Run(ctx context.Context, cfg RunConfig) (*RunResult, error)
	Exec(ctx context.Context, cfg ExecConfig) (*RunResult, error)
	ExecInteractive(ctx context.Context, cfg ExecConfig) error
	ContainerEnv(ctx context.Context, containerName string) (map[string]string, error)
	Pull(ctx context.Context, imageName string) error
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

func (r *SDKRunner) ContainerRunning(ctx context.Context, containerName string) (bool, error) {
	inspect, err := r.client.ContainerInspect(ctx, containerName)
	if err != nil {
		if client.IsErrNotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("inspect container: %w", err)
	}
	return inspect.State != nil && inspect.State.Running, nil
}

func (r *SDKRunner) PublishedHostPort(ctx context.Context, containerName, containerPort string) (string, error) {
	inspect, err := r.client.ContainerInspect(ctx, containerName)
	if err != nil {
		return "", fmt.Errorf("inspect container: %w", err)
	}
	bindings := inspect.NetworkSettings.Ports[nat.Port(containerPort+"/tcp")]
	if len(bindings) == 0 {
		return "", nil
	}
	return bindings[0].HostPort, nil
}

func (r *SDKRunner) ContainerEnv(ctx context.Context, containerName string) (map[string]string, error) {
	inspect, err := r.client.ContainerInspect(ctx, containerName)
	if err != nil {
		return nil, fmt.Errorf("inspect container: %w", err)
	}
	envs := make(map[string]string)
	for _, e := range inspect.Config.Env {
		if k, v, ok := strings.Cut(e, "="); ok {
			envs[k] = v
		}
	}
	return envs, nil
}

func (r *SDKRunner) StreamContainerLogs(ctx context.Context, containerName string, follow bool) error {
	inspect, err := r.client.ContainerInspect(ctx, containerName)
	if err != nil {
		return fmt.Errorf("inspect container: %w", err)
	}
	reader, err := r.client.ContainerLogs(ctx, containerName, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     follow,
	})
	if err != nil {
		return fmt.Errorf("container logs: %w", err)
	}
	defer reader.Close()

	if inspect.Config.Tty {
		_, err = io.Copy(os.Stdout, reader)
		return err
	}
	_, err = stdcopy.StdCopy(os.Stdout, os.Stderr, reader)
	return err
}

func (r *SDKRunner) Exec(ctx context.Context, cfg ExecConfig) (*RunResult, error) {
	execCfg := container.ExecOptions{
		Cmd:          cfg.Cmd,
		WorkingDir:   cfg.WorkingDir,
		User:         cfg.User,
		Env:          cfg.Env,
		AttachStdout: true,
		AttachStderr: true,
	}

	execID, err := r.client.ContainerExecCreate(ctx, cfg.Container, execCfg)
	if err != nil {
		return nil, fmt.Errorf("create exec: %w", err)
	}

	resp, err := r.client.ContainerExecAttach(ctx, execID.ID, container.ExecAttachOptions{})
	if err != nil {
		return nil, fmt.Errorf("attach exec: %w", err)
	}
	defer resp.Close()

	var stdoutBuf, stderrBuf bytes.Buffer
	stdcopy.StdCopy(&stdoutBuf, &stderrBuf, resp.Reader)

	inspect, err := r.client.ContainerExecInspect(ctx, execID.ID)
	if err != nil {
		return nil, fmt.Errorf("inspect exec: %w", err)
	}

	return &RunResult{
		Stdout:   stdoutBuf.String(),
		Stderr:   stderrBuf.String(),
		ExitCode: int64(inspect.ExitCode),
	}, nil
}

func (r *SDKRunner) ExecInteractive(ctx context.Context, cfg ExecConfig) error {
	execCfg := container.ExecOptions{
		Cmd:          cfg.Cmd,
		WorkingDir:   cfg.WorkingDir,
		User:         cfg.User,
		Env:          cfg.Env,
		Tty:          true,
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
	}

	execID, err := r.client.ContainerExecCreate(ctx, cfg.Container, execCfg)
	if err != nil {
		return fmt.Errorf("create exec: %w", err)
	}

	resp, err := r.client.ContainerExecAttach(ctx, execID.ID, container.ExecAttachOptions{Tty: true})
	if err != nil {
		return fmt.Errorf("attach exec: %w", err)
	}
	defer resp.Close()

	inFd := os.Stdin.Fd()
	if term.IsTerminal(inFd) {
		oldState, err := term.MakeRaw(inFd)
		if err != nil {
			return fmt.Errorf("make raw terminal: %w", err)
		}
		defer term.RestoreTerminal(inFd, oldState)
	}

	r.resizeExecTTY(ctx, execID.ID, inFd)

	stopResize := r.monitorResizeEvents(ctx, execID.ID, inFd)
	defer stopResize()

	outputDone := make(chan error, 1)
	go func() {
		_, err := io.Copy(os.Stdout, resp.Reader)
		outputDone <- err
	}()

	go func() {
		io.Copy(resp.Conn, os.Stdin)
		resp.CloseWrite()
	}()

	<-outputDone

	exitCode, err := r.waitForExec(ctx, execID.ID)
	if err != nil {
		return err
	}
	if exitCode == 0 {
		return nil
	}
	return &ExitError{Code: exitCode}
}

func (r *SDKRunner) waitForExec(ctx context.Context, execID string) (int, error) {
	for {
		inspect, err := r.client.ContainerExecInspect(ctx, execID)
		if err != nil {
			return 0, fmt.Errorf("inspect exec: %w", err)
		}
		if !inspect.Running {
			return inspect.ExitCode, nil
		}
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-time.After(50 * time.Millisecond):
		}
	}
}

func (r *SDKRunner) resizeExecTTY(ctx context.Context, execID string, fd uintptr) {
	ws, err := term.GetWinsize(fd)
	if err != nil {
		return
	}
	_ = r.client.ContainerExecResize(ctx, execID, container.ResizeOptions{
		Height: uint(ws.Height),
		Width:  uint(ws.Width),
	})
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

func (r *SDKRunner) Pull(ctx context.Context, imageName string) error {
	reader, err := r.client.ImagePull(ctx, imageName, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("pull image %s: %w", imageName, err)
	}
	defer reader.Close()

	fd, isTerm := term.GetFdInfo(os.Stdout)
	return jsonmessage.DisplayJSONMessagesStream(reader, os.Stdout, fd, isTerm, nil)
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

func (r *SDKRunner) Ping(ctx context.Context) error {
	_, err := r.client.Ping(ctx)
	return err
}

func (r *SDKRunner) NetworkExists(ctx context.Context, name string) (bool, error) {
	_, err := r.client.NetworkInspect(ctx, name, network.InspectOptions{})
	if err == nil {
		return true, nil
	}
	if client.IsErrNotFound(err) {
		return false, nil
	}
	return false, fmt.Errorf("inspect network %s: %w", name, err)
}

func (r *SDKRunner) PublishedPortOwner(ctx context.Context, port string) (string, error) {
	containers, err := r.client.ContainerList(ctx, container.ListOptions{
		Filters: filters.NewArgs(filters.Arg("publish", port)),
	})
	if err != nil {
		return "", fmt.Errorf("list containers publishing %s: %w", port, err)
	}
	if len(containers) == 0 || len(containers[0].Names) == 0 {
		return "", nil
	}
	return strings.TrimPrefix(containers[0].Names[0], "/"), nil
}

func (r *SDKRunner) EnsureNetwork(ctx context.Context, name string) error {
	if _, err := r.client.NetworkInspect(ctx, name, network.InspectOptions{}); err == nil {
		return nil
	} else if !client.IsErrNotFound(err) {
		return fmt.Errorf("inspect network %s: %w", name, err)
	}
	if _, err := r.client.NetworkCreate(ctx, name, network.CreateOptions{}); err != nil {
		return fmt.Errorf("create network %s: %w", name, err)
	}
	return nil
}

func (r *SDKRunner) ProxyInfo(ctx context.Context, networkName string) (bool, uint16, error) {
	containers, err := r.client.ContainerList(ctx, container.ListOptions{
		Filters: filters.NewArgs(filters.Arg("network", networkName)),
	})
	if err != nil {
		return false, 0, fmt.Errorf("list containers on %s: %w", networkName, err)
	}
	for _, c := range containers {
		if !strings.Contains(c.Image, "traefik") {
			continue
		}
		for _, p := range c.Ports {
			if p.PrivatePort == 80 && p.PublicPort != 0 {
				return true, p.PublicPort, nil
			}
		}
		return true, 80, nil
	}
	return false, 0, nil
}

func (r *SDKRunner) ComposeProjects(ctx context.Context, service string) ([]string, error) {
	containers, err := r.client.ContainerList(ctx, container.ListOptions{
		All:     true,
		Filters: filters.NewArgs(filters.Arg("label", "com.docker.compose.service="+service)),
	})
	if err != nil {
		return nil, fmt.Errorf("list %s containers: %w", service, err)
	}
	seen := map[string]bool{}
	var projects []string
	for _, c := range containers {
		project := c.Labels["com.docker.compose.project"]
		if project == "" || seen[project] {
			continue
		}
		seen[project] = true
		projects = append(projects, project)
	}
	return projects, nil
}

type StackSummary struct {
	Project string
	State   string
	Health  string
	URL     string
	Clone   string
	ILS     string
}

func (r *SDKRunner) StackSummaries(ctx context.Context, mainService string) ([]StackSummary, error) {
	containers, err := r.client.ContainerList(ctx, container.ListOptions{
		All:     true,
		Filters: filters.NewArgs(filters.Arg("label", "com.docker.compose.service="+mainService)),
	})
	if err != nil {
		return nil, fmt.Errorf("list %s containers: %w", mainService, err)
	}

	var stacks []StackSummary
	for _, c := range containers {
		project := c.Labels["com.docker.compose.project"]
		if project == "" {
			continue
		}
		inspect, err := r.client.ContainerInspect(ctx, c.ID)
		if err != nil {
			return nil, fmt.Errorf("inspect %s: %w", c.ID, err)
		}
		stacks = append(stacks, summarizeStack(project, inspect))
	}
	sort.Slice(stacks, func(i, j int) bool { return stacks[i].Project < stacks[j].Project })
	return stacks, nil
}

func summarizeStack(project string, inspect container.InspectResponse) StackSummary {
	summary := StackSummary{Project: project, State: inspect.State.Status, ILS: "none"}
	if inspect.State.Health != nil {
		summary.Health = inspect.State.Health.Status
	}
	for _, e := range inspect.Config.Env {
		if v, ok := strings.CutPrefix(e, "URL="); ok {
			summary.URL = v
		}
	}
	for _, m := range inspect.Mounts {
		if m.Destination == "/usr/local/aspen-discovery" {
			summary.Clone = strings.TrimPrefix(m.Source, "/host_mnt")
		}
	}
	for name := range inspect.NetworkSettings.Networks {
		if strings.HasSuffix(name, "_kohanet") {
			summary.ILS = "koha"
		}
		if name == "evergreen-net" {
			summary.ILS = "evergreen"
		}
	}
	return summary
}

func (r *SDKRunner) ProxiedStacks(ctx context.Context, proxyProject string) (int, error) {
	containers, err := r.client.ContainerList(ctx, container.ListOptions{
		Filters: filters.NewArgs(filters.Arg("label", "aspen.proxy=true")),
	})
	if err != nil {
		return 0, fmt.Errorf("list proxied containers: %w", err)
	}
	seen := map[string]bool{}
	for _, c := range containers {
		project := c.Labels["com.docker.compose.project"]
		if project != proxyProject {
			seen[project] = true
		}
	}
	return len(seen), nil
}

func (r *SDKRunner) RemoveProjectVolumes(ctx context.Context, project string) error {
	vols, err := r.client.VolumeList(ctx, volume.ListOptions{
		Filters: filters.NewArgs(filters.Arg("label", "com.docker.compose.project="+project)),
	})
	if err != nil {
		return fmt.Errorf("list %s volumes: %w", project, err)
	}
	for _, v := range vols.Volumes {
		if err := r.client.VolumeRemove(ctx, v.Name, false); err != nil {
			return fmt.Errorf("remove volume %s: %w", v.Name, err)
		}
	}
	return nil
}
