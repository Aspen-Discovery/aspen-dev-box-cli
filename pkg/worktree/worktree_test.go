package worktree

import "testing"

const porcelain = `worktree /Users/dev/aspen-discovery
HEAD 1111111111111111111111111111111111111111
branch refs/heads/main

worktree /Users/dev/aspen-discovery/worktrees/26/omeka-integration
HEAD 2222222222222222222222222222222222222222
branch refs/heads/feature/omeka_integration

worktree /Users/dev/aspen-detached
HEAD 3333333333333333333333333333333333333333
detached

`

func TestParsePorcelain(t *testing.T) {
	trees := parsePorcelain(porcelain)
	if len(trees) != 3 {
		t.Fatalf("expected 3 worktrees, got %d: %+v", len(trees), trees)
	}

	main := trees[0]
	if main.Name != "aspen-discovery" || main.Branch != "main" || main.Path != "/Users/dev/aspen-discovery" {
		t.Errorf("unexpected main worktree: %+v", main)
	}

	feature := trees[1]
	if feature.Name != "omeka-integration" || feature.Branch != "feature/omeka_integration" {
		t.Errorf("unexpected feature worktree: %+v", feature)
	}
	if feature.SHA != "2222222222222222222222222222222222222222" {
		t.Errorf("SHA not parsed: %+v", feature)
	}

	detached := trees[2]
	if detached.Branch != "" || detached.Name != "aspen-detached" {
		t.Errorf("detached worktree should have no branch: %+v", detached)
	}
}

func TestParsePorcelainWithoutTrailingBlankLine(t *testing.T) {
	trees := parsePorcelain("worktree /repo\nHEAD 1111\nbranch refs/heads/main")
	if len(trees) != 1 || trees[0].Branch != "main" {
		t.Errorf("last worktree without trailing newline dropped: %+v", trees)
	}
}

func TestSafeName(t *testing.T) {
	cases := map[string]string{
		"feature/omeka_integration": "feature-omeka-integration",
		"Release.26.09":             "release-26-09",
		"/leading-and-trailing/":    "leading-and-trailing",
		"plain":                     "plain",
	}
	for in, want := range cases {
		if got := SafeName(in); got != want {
			t.Errorf("SafeName(%q) = %q, want %q", in, got, want)
		}
	}
}
