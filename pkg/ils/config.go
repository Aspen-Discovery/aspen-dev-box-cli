package ils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Base            string         `yaml:"base"`
	Driver          string         `yaml:"driver"`
	AccountProfile  map[string]any `yaml:"account_profile"`
	IndexingProfile map[string]any `yaml:"indexing_profile"`
	ExtrasSQL       string         `yaml:"extras_sql"`
}

func Load(path string) (*Config, error) {
	cfg, err := loadOne(path)
	if err != nil {
		return nil, err
	}
	if cfg.Base == "" {
		return cfg, nil
	}
	basePath := filepath.Join(filepath.Dir(path), cfg.Base)
	base, err := Load(basePath)
	if err != nil {
		return nil, fmt.Errorf("load base %s: %w", basePath, err)
	}
	return merge(base, cfg), nil
}

func loadOne(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read ils config %s: %w", path, err)
	}
	expanded := os.ExpandEnv(string(data))
	var cfg Config
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return nil, fmt.Errorf("parse ils config %s: %w", path, err)
	}
	return &cfg, nil
}

func merge(base, override *Config) *Config {
	out := *override
	if out.Driver == "" {
		out.Driver = base.Driver
	}
	if out.ExtrasSQL == "" {
		out.ExtrasSQL = base.ExtrasSQL
	}
	out.IndexingProfile = mergeMap(base.IndexingProfile, override.IndexingProfile)
	return &out
}

func mergeMap(base, override map[string]any) map[string]any {
	out := make(map[string]any, len(base)+len(override))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range override {
		out[k] = v
	}
	return out
}

func ResolvePath(value, presetDir string) (string, error) {
	if value == "" || value == "none" {
		return "", nil
	}
	if strings.HasPrefix(value, "_") {
		return "", fmt.Errorf("ils preset %q is reserved (leading underscore)", value)
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
