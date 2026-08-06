package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

// Docker compose files, relative to the compose/ directory
const (
	DefaultComposeFile   = "docker-compose.yml"
	DebugComposeFile     = "docker-compose.debug.yml"
	JavaDebugComposeFile = "docker-compose.debug.java.yml"
	DBGUIComposeFile     = "docker-compose.dbgui.yml"
	ILSComposeFile       = "docker-compose.ils.yml"
	KohaComposeFile      = "docker-compose.koha.yml"
	EvergreenComposeFile = "docker-compose.evergreen.yml"
	PluginsComposeFile   = "docker-compose.plugins.yml"
	TestsComposeFile     = "docker-compose.tests.yml"
	ProxyComposeFile     = "docker-compose.proxy.yml"
)

// Config holds all application configuration
type Config struct {
	// Required - from environment
	ProjectsDir   string
	AspenCloneDir string

	StackName            string
	MainContainerService string
	MainContainerWorkDir string
	DBContainerService   string

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
		MainContainerService: "aspen-dev-box",
		MainContainerWorkDir: "/usr/local/aspen-discovery",
		DBContainerService:   "aspen-db",
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

	cfg.StackName = resolveStackName(cfg.ProjectsDir)

	return cfg, nil
}

// UseWorktree points the config at a worktree checkout: it becomes the aspen
// clone and the given stack name takes over unless --stack was set, so every
// command targets that instance.
func (c *Config) UseWorktree(path, stack string, stackOverridden bool) {
	c.AspenCloneDir = path
	if !stackOverridden {
		c.StackName = stack
	}
	os.Setenv("ASPEN_CLONE", path)
}

func resolveStackName(projectsDir string) string {
	if v := os.Getenv("ASPEN_STACK"); v != "" {
		return v
	}
	if v := os.Getenv("COMPOSE_PROJECT_NAME"); v != "" {
		return v
	}
	return filepath.Base(projectsDir)
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
	return filepath.Join(c.ProjectsDir, "compose", filename)
}

// ILSSQLPath returns the generated ILS setup SQL path for a stack. Scoped
// per stack: the file is bind-mounted into the stack's db container and read
// whenever that db (re)initialises, so stacks must not share it.
func (c *Config) ILSSQLPath(stack string) string {
	return filepath.Join(c.ProjectsDir, ".cache", stack+"-ils-setup.sql")
}

// DBConnectionString returns the mariadb connection string
func (c *Config) DBConnectionString() string {
	return fmt.Sprintf("-u%s -p%s %s", c.DBUser, c.DBPassword, c.DBName)
}

func (c *Config) ContainerName(service string) string {
	return fmt.Sprintf("%s-%s-1", c.StackName, service)
}

func (c *Config) MainContainerName() string {
	return c.ContainerName(c.MainContainerService)
}

func (c *Config) DBContainerName() string {
	return c.ContainerName(c.DBContainerService)
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
