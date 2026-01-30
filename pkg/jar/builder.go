package jar

import (
	"context"
	_ "embed"
	"fmt"
	"os"

	"adb/pkg/docker"
)

//go:embed scripts/build.sh
var buildScript string

type BuildConfig struct {
	AspenCloneDir   string
	JavaImage       string
	SharedLibsPath  string
	ExcludePatterns []string
}

type Builder struct {
	config BuildConfig
	runner docker.Runner
}

func NewBuilder(cfg BuildConfig, runner docker.Runner) *Builder {
	return &Builder{
		config: cfg,
		runner: runner,
	}
}

func (b *Builder) Build(ctx context.Context, module Module) error {
	fmt.Printf("\n\033[1;34mRecompiling JAR file: %s\033[0m\n", module.Name)

	result, err := b.runner.Run(ctx, docker.RunConfig{
		Image:      b.config.JavaImage,
		Cmd:        []string{"bash", "-c", buildScript},
		WorkingDir: fmt.Sprintf("/app/code/%s", module.Name),
		User:       fmt.Sprintf("%d:%d", os.Getuid(), os.Getgid()),
		Binds:      []string{fmt.Sprintf("%s:/app", b.config.AspenCloneDir)},
		Env:        []string{fmt.Sprintf("SHARED_LIBS_PATH=%s", b.config.SharedLibsPath)},
	})
	if err != nil {
		return fmt.Errorf("build %s: %w", module.Name, err)
	}

	fmt.Print(result.Stdout)
	if result.Stderr != "" {
		fmt.Fprint(os.Stderr, result.Stderr)
	}

	if result.ExitCode != 0 {
		return fmt.Errorf("build failed with exit code %d", result.ExitCode)
	}

	return nil
}

func (b *Builder) BuildAll(ctx context.Context) error {
	codeDir := fmt.Sprintf("%s/code", b.config.AspenCloneDir)

	modules, err := DiscoverModules(codeDir, b.config.ExcludePatterns)
	if err != nil {
		return fmt.Errorf("discover modules: %w", err)
	}

	for _, module := range modules {
		if err := b.Build(ctx, module); err != nil {
			return err
		}
	}

	return nil
}

func (b *Builder) GetModuleNames(ctx context.Context) ([]string, error) {
	codeDir := fmt.Sprintf("%s/code", b.config.AspenCloneDir)

	modules, err := DiscoverModules(codeDir, b.config.ExcludePatterns)
	if err != nil {
		return nil, err
	}

	names := make([]string, len(modules))
	for i, m := range modules {
		names[i] = m.Name
	}

	return names, nil
}
