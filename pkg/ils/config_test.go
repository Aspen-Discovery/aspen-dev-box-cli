package ils

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadMergesBase(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "_base.yml"), `
indexing_profile:
  marcEncoding: UTF8
  formatSource: item
  dateCreatedFormat: yyyy-MM-dd
`)
	writeFile(t, filepath.Join(dir, "koha.yml"), `
base: _base.yml
driver: Koha
account_profile:
  name: ils
  driver: Koha
indexing_profile:
  indexingClass: Koha
`)

	cfg, err := Load(filepath.Join(dir, "koha.yml"))
	if err != nil {
		t.Fatal(err)
	}

	if cfg.IndexingProfile["marcEncoding"] != "UTF8" {
		t.Errorf("base field missing: %v", cfg.IndexingProfile)
	}
	if cfg.IndexingProfile["indexingClass"] != "Koha" {
		t.Errorf("override missing: %v", cfg.IndexingProfile)
	}
	if cfg.AccountProfile["name"] != "ils" {
		t.Errorf("account profile not loaded: %v", cfg.AccountProfile)
	}
}

func TestLoadOverrideWinsOnConflict(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "_base.yml"), `
indexing_profile:
  marcEncoding: ASCII
`)
	writeFile(t, filepath.Join(dir, "preset.yml"), `
base: _base.yml
indexing_profile:
  marcEncoding: UTF8
`)

	cfg, err := Load(filepath.Join(dir, "preset.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.IndexingProfile["marcEncoding"] != "UTF8" {
		t.Errorf("override should win, got: %v", cfg.IndexingProfile["marcEncoding"])
	}
}

func TestAccountProfileNotInherited(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "_base.yml"), `
account_profile:
  databaseHost: should-not-leak
`)
	writeFile(t, filepath.Join(dir, "preset.yml"), `
base: _base.yml
account_profile:
  name: ils
`)

	cfg, err := Load(filepath.Join(dir, "preset.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := cfg.AccountProfile["databaseHost"]; ok {
		t.Errorf("account_profile leaked from base: %v", cfg.AccountProfile)
	}
}

func TestRenderSQLIncludesMergedFields(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "_base.yml"), `
indexing_profile:
  marcEncoding: UTF8
`)
	writeFile(t, filepath.Join(dir, "preset.yml"), `
base: _base.yml
driver: Koha
account_profile:
  name: ils
indexing_profile:
  indexingClass: Koha
`)

	cfg, err := Load(filepath.Join(dir, "preset.yml"))
	if err != nil {
		t.Fatal(err)
	}
	sql, err := cfg.RenderSQL(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(sql, "marcEncoding") || !strings.Contains(sql, "indexingClass") {
		t.Errorf("expected merged indexing fields in SQL:\n%s", sql)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
