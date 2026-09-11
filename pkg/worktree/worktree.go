package worktree

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

type Worktree struct {
	Name   string
	Path   string
	Branch string
	SHA    string
}

func SafeName(ref string) string {
	name := strings.ToLower(ref)
	name = strings.NewReplacer("/", "-", "_", "-", ".", "-").Replace(name)
	return strings.Trim(name, "-")
}

func Find(ctx context.Context, repo, name string) (Worktree, error) {
	trees, err := List(ctx, repo)
	if err != nil {
		return Worktree{}, err
	}
	for _, t := range trees {
		matches := t.Name == name || t.Branch == name || SafeName(t.Branch) == name
		if matches {
			return t, nil
		}
	}
	names := make([]string, len(trees))
	for i, t := range trees {
		names[i] = t.Name
	}
	return Worktree{}, fmt.Errorf("no worktree %q in %s (have: %s; create one with: git -C $ASPEN_CLONE worktree add <path> <ref>)",
		name, repo, strings.Join(names, ", "))
}

func List(ctx context.Context, repo string) ([]Worktree, error) {
	out, err := output(ctx, repo, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}
	return parsePorcelain(out), nil
}

func parsePorcelain(out string) []Worktree {
	var trees []Worktree
	var current Worktree
	for _, line := range strings.Split(out, "\n") {
		switch {
		case strings.HasPrefix(line, "worktree "):
			current = Worktree{Path: strings.TrimPrefix(line, "worktree ")}
			current.Name = filepath.Base(current.Path)
		case strings.HasPrefix(line, "HEAD "):
			current.SHA = strings.TrimPrefix(line, "HEAD ")
		case strings.HasPrefix(line, "branch "):
			current.Branch = strings.TrimPrefix(strings.TrimPrefix(line, "branch "), "refs/heads/")
		case line == "":
			if current.Path != "" {
				trees = append(trees, current)
			}
			current = Worktree{}
		}
	}
	if current.Path != "" {
		trees = append(trees, current)
	}
	return trees
}

func output(ctx context.Context, repo string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", repo}, args...)...)
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), strings.TrimSpace(string(ee.Stderr)))
		}
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return string(out), nil
}
