package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

// Config holds all application configuration
type Config struct {
	// Required - from environment
	ProjectsDir   string
	AspenCloneDir string

	// Docker compose files
	DefaultComposeFile   string
	DebugComposeFile     string
	DBGUIComposeFile     string
	EvergreenComposeFile string

	// Container settings
	MainContainerName    string
	MainContainerWorkDir string
	DBContainerName      string

	// Database settings
	DBName     string
	DBUser     string
	DBPassword string

	// Paths
	LogPath            string
	JSWorkDir          string
	CSSBaseDir         string
	JavaSharedLibsPath string

	// Docker images
	JavaBuildImage string
	AlpineImage    string
	LessImage      string

	// Build settings
	ExcludedJarPatterns []string
	MergeJSScript       string
	LessInputFile       string
	LessOutputFile      string
}

// Load reads configuration from environment and .env file
func Load() (*Config, error) {
	if err := loadEnvFile(); err != nil {
		return nil, err
	}

	cfg := &Config{
		// Defaults
		DefaultComposeFile:   "docker-compose.yml",
		DebugComposeFile:     "docker-compose.debug.yml",
		DBGUIComposeFile:     "docker-compose.dbgui.yml",
		EvergreenComposeFile: "docker-compose.evergreen.yml",
		MainContainerName:    "containeraspen",
		MainContainerWorkDir: "/usr/local/aspen-discovery",
		DBContainerName:      "aspen-db",
		DBName:               "aspen",
		DBUser:               "root",
		DBPassword:           "aspen",
		LogPath:              "/var/log/aspen-discovery/test.localhostaspen/",
		JSWorkDir:            "/usr/local/aspen-discovery/code/web/interface/themes/responsive/js",
		CSSBaseDir:           "/code/web/interface/themes/responsive/css",
		JavaSharedLibsPath:   "/app/code/java_shared_libraries",
		JavaBuildImage:       "adoptopenjdk:11",
		AlpineImage:          "alpine:latest",
		LessImage:            "ghcr.io/sndsgd/less",
		ExcludedJarPatterns:  []string{"java_shared_libraries"},
		MergeJSScript:        "merge_javascript.php",
		LessInputFile:        "main.less",
		LessOutputFile:       "main.css",
	}

	// Required environment variables
	cfg.ProjectsDir = os.Getenv("ASPEN_DOCKER")
	if cfg.ProjectsDir == "" {
		return nil, fmt.Errorf("ASPEN_DOCKER environment variable not set")
	}

	cfg.AspenCloneDir = os.Getenv("ASPEN_CLONE")
	if cfg.AspenCloneDir == "" {
		return nil, fmt.Errorf("ASPEN_CLONE environment variable not set")
	}

	return cfg, nil
}

// loadEnvFile attempts to load .env file relative to binary location
func loadEnvFile() error {
	ex, err := os.Executable()
	if err != nil {
		return nil // Not fatal - env vars might be set directly
	}

	binaryDir := filepath.Dir(ex)
	// binary is in folder/bin/architecture/binary, .env is in folder/.env
	envPath := filepath.Join(filepath.Dir(filepath.Dir(binaryDir)), ".env")

	if err := godotenv.Load(envPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("load .env file: %w", err)
	}
	return nil
}

func (c *Config) ApplyContainerEnv(env map[string]string) {
	if v, ok := env["SITE_NAME"]; ok {
		c.LogPath = "/var/log/aspen-discovery/" + v + "/"
	}
	if v, ok := env["DATABASE_NAME"]; ok {
		c.DBName = v
	}
	if v, ok := env["DATABASE_USER"]; ok {
		c.DBUser = v
	}
	if v, ok := env["DATABASE_PASSWORD"]; ok {
		c.DBPassword = v
	}
}

// ComposeFilePath returns full path to a compose file
func (c *Config) ComposeFilePath(filename string) string {
	return filepath.Join(c.ProjectsDir, filename)
}

// DefaultComposeFilePath returns path to the default docker-compose file
func (c *Config) DefaultComposeFilePath() string {
	return c.ComposeFilePath(c.DefaultComposeFile)
}

// DebugComposeFilePath returns path to the debug docker-compose file
func (c *Config) DebugComposeFilePath() string {
	return c.ComposeFilePath(c.DebugComposeFile)
}

// DBGUIComposeFilePath returns path to the dbgui docker-compose file
func (c *Config) DBGUIComposeFilePath() string {
	return c.ComposeFilePath(c.DBGUIComposeFile)
}

// EvergreenComposeFilePath returns path to the evergreen docker-compose file
func (c *Config) EvergreenComposeFilePath() string {
	return c.ComposeFilePath(c.EvergreenComposeFile)
}

// DBConnectionString returns the mariadb connection string
func (c *Config) DBConnectionString() string {
	return fmt.Sprintf("-u%s -p%s %s", c.DBUser, c.DBPassword, c.DBName)
}

// CSSDir returns the path to CSS directory, with optional RTL suffix
func (c *Config) CSSDir(rtl bool) string {
	dir := filepath.Join(c.AspenCloneDir, c.CSSBaseDir)
	if rtl {
		dir += "-rtl"
	}
	return dir
}

// CodeDir returns the path to the code directory
func (c *Config) CodeDir() string {
	return filepath.Join(c.AspenCloneDir, "code")
}
