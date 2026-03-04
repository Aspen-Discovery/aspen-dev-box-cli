//go:build windows

package docker

import "context"

func (r *SDKRunner) monitorResizeEvents(_ context.Context, _ string, _ uintptr) func() {
	return func() {}
}
