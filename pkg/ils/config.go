package ils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Driver          string         `yaml:"driver"`
	AccountProfile  map[string]any `yaml:"account_profile"`
	IndexingProfile map[string]any `yaml:"indexing_profile"`
	ExtrasSQL       string         `yaml:"extras_sql"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read ils config %s: %w", path, err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse ils config %s: %w", path, err)
	}
	return &cfg, nil
}

func ResolvePath(value, presetDir string) (string, error) {
	if value == "" || value == "none" {
		return "", nil
	}
	if looksLikePath(value) {
		abs, err := filepath.Abs(value)
		if err != nil {
			return "", err
		}
		if _, err := os.Stat(abs); err != nil {
			return "", fmt.Errorf("ils config not found at %s", abs)
		}
		return abs, nil
	}
	preset := filepath.Join(presetDir, value+".yml")
	if _, err := os.Stat(preset); err != nil {
		return "", fmt.Errorf("ils preset %q not found at %s", value, preset)
	}
	return preset, nil
}

func looksLikePath(v string) bool {
	if strings.ContainsAny(v, "/\\") {
		return true
	}
	return strings.HasSuffix(v, ".yml") || strings.HasSuffix(v, ".yaml")
}
