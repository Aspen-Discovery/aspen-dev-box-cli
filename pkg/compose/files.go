package compose

import (
	"fmt"
	"os"
)

// Options holds the configuration for selecting compose files
type Options struct {
	Debugging bool
	DBGui     bool
}

// GetComposeFiles returns a list of compose file paths based on the provided options
func GetComposeFiles(opts Options) ([]string, error) {
	projectsDir := os.Getenv("ASPEN_DOCKER")
	if projectsDir == "" {
		return nil, fmt.Errorf("ASPEN_DOCKER environment variable not set")
	}

	files := []string{
		fmt.Sprintf("%s/docker-compose.yml", projectsDir),
	}

	if opts.Debugging {
		files = append(files, fmt.Sprintf("%s/docker-compose.debug.yml", projectsDir))
	}

	if opts.DBGui {
		files = append(files, fmt.Sprintf("%s/docker-compose.dbgui.yml", projectsDir))
	}

	return files, nil
}
