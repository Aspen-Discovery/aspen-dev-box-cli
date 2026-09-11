package config

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestLoadReadsEnvFileFromProjectsDir(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("ADB_TEST_FROM_ENV_FILE=loaded\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ASPEN_DOCKER", dir)
	t.Setenv("ASPEN_CLONE", t.TempDir())
	t.Setenv("ADB_TEST_FROM_ENV_FILE", "")
	os.Unsetenv("ADB_TEST_FROM_ENV_FILE")

	if _, err := Load(); err != nil {
		t.Fatal(err)
	}
	if got := os.Getenv("ADB_TEST_FROM_ENV_FILE"); got != "loaded" {
		t.Errorf("expected .env under ASPEN_DOCKER to be loaded, got %q", got)
	}
}

func TestLoadDoesNotOverrideExistingEnv(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("ADB_TEST_PRESET=file\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ASPEN_DOCKER", dir)
	t.Setenv("ASPEN_CLONE", t.TempDir())
	t.Setenv("ADB_TEST_PRESET", "shell")

	if _, err := Load(); err != nil {
		t.Fatal(err)
	}
	if got := os.Getenv("ADB_TEST_PRESET"); got != "shell" {
		t.Errorf("shell env should win over .env, got %q", got)
	}
}

func TestLoadExportsHostIDsWhenUnset(t *testing.T) {
	if os.Getuid() < 0 {
		t.Skip("no numeric uid on this platform")
	}
	t.Setenv("ASPEN_DOCKER", t.TempDir())
	t.Setenv("ASPEN_CLONE", t.TempDir())
	t.Setenv("UID", "")
	t.Setenv("GID", "")
	os.Unsetenv("UID")
	os.Unsetenv("GID")

	if _, err := Load(); err != nil {
		t.Fatal(err)
	}
	if got := os.Getenv("UID"); got != strconv.Itoa(os.Getuid()) {
		t.Errorf("UID = %q, want %d", got, os.Getuid())
	}
	if got := os.Getenv("GID"); got != strconv.Itoa(os.Getgid()) {
		t.Errorf("GID = %q, want %d", got, os.Getgid())
	}
}

func TestLoadKeepsExportedHostIDs(t *testing.T) {
	t.Setenv("ASPEN_DOCKER", t.TempDir())
	t.Setenv("ASPEN_CLONE", t.TempDir())
	t.Setenv("UID", "4242")
	t.Setenv("GID", "4343")

	if _, err := Load(); err != nil {
		t.Fatal(err)
	}
	if os.Getenv("UID") != "4242" || os.Getenv("GID") != "4343" {
		t.Errorf("exported UID/GID must win, got %s/%s", os.Getenv("UID"), os.Getenv("GID"))
	}
}

func TestLoadRequiresAspenDocker(t *testing.T) {
	t.Setenv("ASPEN_DOCKER", "")
	t.Setenv("ASPEN_CLONE", t.TempDir())
	if _, err := Load(); err == nil {
		t.Error("expected an error when ASPEN_DOCKER is unset")
	}
}
