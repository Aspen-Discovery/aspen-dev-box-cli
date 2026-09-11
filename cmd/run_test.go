package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverJobsFindsBuiltJarsAndPHPJobs(t *testing.T) {
	codeDir := t.TempDir()
	mustMkdir(t, filepath.Join(codeDir, "reindexer"))
	mustWrite(t, filepath.Join(codeDir, "reindexer", "reindexer.jar"))
	mustMkdir(t, filepath.Join(codeDir, "web"))
	mustMkdir(t, filepath.Join(codeDir, "unbuilt_export"))
	mustWrite(t, filepath.Join(codeDir, "IntelliJCodeStyle.xml"))

	jobs, err := discoverJobs(codeDir)
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := jobs["reindexer"]; !ok {
		t.Errorf("module with a built jar should be a job: %v", sortedJobs(jobs))
	}
	if _, ok := jobs["unbuilt_export"]; ok {
		t.Error("module without a jar must not be a job")
	}
	if _, ok := jobs["web"]; ok {
		t.Error("web has no jar and must not be a job")
	}
	for _, name := range []string{"cron", "sitemaps"} {
		if _, ok := jobs[name]; !ok {
			t.Errorf("php job %s missing", name)
		}
	}

	job := jobs["reindexer"]
	if job.Dir != "/usr/local/aspen-discovery/code/reindexer" || job.Bin[len(job.Bin)-1] != "reindexer.jar" {
		t.Errorf("unexpected jar job: %+v", job)
	}
}

func TestDiscoverJobsFailsOnMissingCodeDir(t *testing.T) {
	if _, err := discoverJobs(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Error("expected an error for a missing code dir")
	}
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
}

func mustWrite(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
}
