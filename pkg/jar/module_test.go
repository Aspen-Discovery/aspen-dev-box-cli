package jar

import (
	"os"
	"path/filepath"
	"testing"
)

func TestManifestMainClass(t *testing.T) {
	manifest := "Manifest-Version: 1.0\r\nMain-Class: org.aspen_discovery.reindexer.GroupedReindexMain\r\nClass-Path: lib/a.jar\r\n"
	if got := manifestMainClass(manifest); got != "org.aspen_discovery.reindexer.GroupedReindexMain" {
		t.Errorf("got %q", got)
	}
	if got := manifestMainClass("Manifest-Version: 1.0\n"); got != "" {
		t.Errorf("expected empty main class, got %q", got)
	}
}

func TestDiscoverModulesUsesEitherManifestLocation(t *testing.T) {
	codeDir := t.TempDir()
	writeManifest(t, filepath.Join(codeDir, "root_manifest", "META-INF"), "a.Main")
	writeManifest(t, filepath.Join(codeDir, "src_manifest", "src", "META-INF"), "b.Main")
	writeManifest(t, filepath.Join(codeDir, "java_shared_libraries", "META-INF"), "c.Main")
	if err := os.MkdirAll(filepath.Join(codeDir, "web"), 0o755); err != nil {
		t.Fatal(err)
	}

	modules, err := DiscoverModules(codeDir, []string{"java_shared_libraries"})
	if err != nil {
		t.Fatal(err)
	}

	byName := map[string]Module{}
	for _, m := range modules {
		byName[m.Name] = m
	}
	if len(byName) != 2 {
		t.Fatalf("expected root_manifest and src_manifest only, got %v", byName)
	}
	if byName["root_manifest"].ManifestPath != "META-INF/MANIFEST.MF" {
		t.Errorf("root manifest path wrong: %+v", byName["root_manifest"])
	}
	if byName["src_manifest"].ManifestPath != "src/META-INF/MANIFEST.MF" {
		t.Errorf("src manifest path wrong: %+v", byName["src_manifest"])
	}

	mainClass, err := byName["src_manifest"].MainClass()
	if err != nil || mainClass != "b.Main" {
		t.Errorf("MainClass = %q, %v", mainClass, err)
	}
}

func TestFindModuleErrors(t *testing.T) {
	codeDir := t.TempDir()
	if _, err := FindModule(codeDir, "nope"); err == nil {
		t.Error("expected error for a missing module")
	}
	if err := os.MkdirAll(filepath.Join(codeDir, "nomanifest", "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := FindModule(codeDir, "nomanifest"); err == nil {
		t.Error("expected error for a module without a manifest")
	}
}

func writeManifest(t *testing.T, dir, mainClass string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := "Manifest-Version: 1.0\nMain-Class: " + mainClass + "\n"
	if err := os.WriteFile(filepath.Join(dir, "MANIFEST.MF"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
