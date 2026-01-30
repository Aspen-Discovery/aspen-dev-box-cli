package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

const (
	// DefaultDockerComposeFile is the default docker-compose file name
	DefaultDockerComposeFile = "docker-compose.yml"
	// DebugDockerComposeFile is the debugging docker-compose file name
	DebugDockerComposeFile = "docker-compose.debug.yml"
	// DBGUIDockerComposeFile is the dbgui docker-compose file name
	DBGUIDockerComposeFile = "docker-compose.dbgui.yml"
	// MainContainerName is the name of the main container
	MainContainerName = "containeraspen"
	// MainContainerWorkDir is the working directory inside the main container
	MainContainerWorkDir = "/usr/local/aspen-discovery"
	// DBContainerName is the name of the database container
	DBContainerName = "aspen-db"
	// DBName is the name of the database
	DBName = "aspen"
	// DBUser is the database username
	DBUser = "root"
	// DBPassword is the database password
	DBPassword = "aspen"
	// LogPath is the path to the logs directory
	LogPath = "/var/log/aspen-discovery/test.localhostaspen/"
	// SupportedShells is a list of supported shells for completion
	SupportedShells = "bash zsh fish"
	// JavaBuildImage is the Docker image used for Java builds
	JavaBuildImage = "adoptopenjdk:11"
	// AlpineImage is the Docker image used for file operations
	AlpineImage = "alpine:latest"
	// JavaSharedLibrariesPath is the path to Java shared libraries
	JavaSharedLibrariesPath = "/app/code/java_shared_libraries"
	// ExcludedJarPatterns contains modules that shouldn't be built as JARs (shared libraries only)
	ExcludedJarPatterns = "java_shared_libraries"
	// JSWorkDir is the working directory for JavaScript operations
	JSWorkDir = "/usr/local/aspen-discovery/code/web/interface/themes/responsive/js"
	// MergeJSScript is the name of the JavaScript merge script
	MergeJSScript = "merge_javascript.php"
	// CSSBaseDir is the base directory for CSS files
	CSSBaseDir = "/code/web/interface/themes/responsive/css"
	// CSSRTLSuffix is the suffix for RTL CSS directory
	CSSRTLSuffix = "-rtl"
	// LessImage is the Docker image used for LESS compilation
	LessImage = "ghcr.io/sndsgd/less"
	// LessInputFile is the input LESS file
	LessInputFile = "main.less"
	// LessOutputFile is the output CSS file
	LessOutputFile = "main.css"
)

// getBinaryPath returns the absolute path to the running binary
func getBinaryPath() (string, error) {
	ex, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to get executable path: %w", err)
	}
	return filepath.Dir(ex), nil
}

// loadEnvFile attempts to load the .env file relative to the binary location
func loadEnvFile() error {
	binaryPath, err := getBinaryPath()
	if err != nil {
		return err
	}

	// Go up two directories from the binary location to find the .env file
	// binary is in folder/bin/architecture/binary, .env is in folder/.env
	envPath := filepath.Join(filepath.Dir(filepath.Dir(binaryPath)), ".env")

	// Try to load the .env file, but don't error if it doesn't exist
	err = godotenv.Load(envPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("error loading .env file: %w", err)
	}
	return nil
}

// GetProjectsDir returns the ASPEN_DOCKER environment variable or falls back to .env file
func GetProjectsDir() string {
	// Try to load .env file first
	_ = loadEnvFile()

	projectsDir := os.Getenv("ASPEN_DOCKER")
	if projectsDir == "" {
		fmt.Println("Error: ASPEN_DOCKER environment variable not set.")
		os.Exit(1)
	}
	return projectsDir
}

// GetAspenCloneDir returns the ASPEN_CLONE environment variable or falls back to .env file
func GetAspenCloneDir() string {
	// Try to load .env file first
	_ = loadEnvFile()

	aspenClone := os.Getenv("ASPEN_CLONE")
	if aspenClone == "" {
		fmt.Println("Error: ASPEN_CLONE environment variable not set.")
		os.Exit(1)
	}
	return aspenClone
}

// GetComposeFilePath returns the full path to a docker-compose file
func GetComposeFilePath(filename string) string {
	return filepath.Join(GetProjectsDir(), filename)
}

// GetDefaultComposeFile returns the path to the default docker-compose file
func GetDefaultComposeFile() string {
	return GetComposeFilePath(DefaultDockerComposeFile)
}

// GetDebugComposeFile returns the path to the debug docker-compose file
func GetDebugComposeFile() string {
	return GetComposeFilePath(DebugDockerComposeFile)
}

// GetDBGUIComposeFile returns the path to the dbgui docker-compose file
func GetDBGUIComposeFile() string {
	return GetComposeFilePath(DBGUIDockerComposeFile)
}

// GetMainContainerName returns the name of the main container
func GetMainContainerName() string {
	return MainContainerName
}

// GetMainContainerWorkDir returns the working directory inside the main container
func GetMainContainerWorkDir() string {
	return MainContainerWorkDir
}

// GetDBContainerName returns the name of the database container
func GetDBContainerName() string {
	return DBContainerName
}

// GetDBConnectionString returns the connection string for the database
func GetDBConnectionString() string {
	return fmt.Sprintf("-u%s -p%s %s", DBUser, DBPassword, DBName)
}

// GetLogPath returns the path to the logs directory
func GetLogPath() string {
	return LogPath
}

// GetSupportedShells returns the list of supported shells
func GetSupportedShells() string {
	return SupportedShells
}

// ValidateShell validates if the given shell is supported
func ValidateShell(shell string) bool {
	switch shell {
	case "bash", "zsh", "fish":
		return true
	default:
		return false
	}
}

// GetJavaBuildImage returns the Docker image used for Java builds
func GetJavaBuildImage() string {
	return JavaBuildImage
}

// GetAlpineImage returns the Docker image used for file operations
func GetAlpineImage() string {
	return AlpineImage
}

// GetJavaSharedLibrariesPath returns the path to Java shared libraries
func GetJavaSharedLibrariesPath() string {
	return JavaSharedLibrariesPath
}

// GetExcludedJarPatterns returns the list of patterns to exclude from JAR builds
func GetExcludedJarPatterns() string {
	return ExcludedJarPatterns
}

// GetJSWorkDir returns the working directory for JavaScript operations
func GetJSWorkDir() string {
	return JSWorkDir
}

// GetMergeJSScript returns the name of the JavaScript merge script
func GetMergeJSScript() string {
	return MergeJSScript
}

// GetCSSDir returns the path to the CSS directory
func GetCSSDir(rtl bool) string {
	dir := filepath.Join(GetAspenCloneDir(), CSSBaseDir)
	if rtl {
		dir += CSSRTLSuffix
	}
	return dir
}

// GetLessImage returns the Docker image used for LESS compilation
func GetLessImage() string {
	return LessImage
}

// GetLessInputFile returns the input LESS file
func GetLessInputFile() string {
	return LessInputFile
}

// GetLessOutputFile returns the output CSS file
func GetLessOutputFile() string {
	return LessOutputFile
}
