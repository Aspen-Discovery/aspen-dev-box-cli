//go:build !windows

package docker

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

func (r *SDKRunner) monitorResizeEvents(ctx context.Context, execID string, fd uintptr) func() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGWINCH)
	go func() {
		for range sigCh {
			r.resizeExecTTY(ctx, execID, fd)
		}
	}()
	return func() { signal.Stop(sigCh) }
}
