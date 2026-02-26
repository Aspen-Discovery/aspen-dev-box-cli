package jar

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Module struct {
	Name           string
	Path           string
	ManifestPath   string
	NeedsSharedLib bool
}

func DiscoverModules(codeDir string, excludePatterns []string) ([]Module, error) {
	entries, err := os.ReadDir(codeDir)
	if err != nil {
		return nil, fmt.Errorf("read code directory: %w", err)
	}

	var modules []Module
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()
		if isExcluded(name, excludePatterns) {
			continue
		}

		modulePath := filepath.Join(codeDir, name)

		module, err := analyzeModule(name, modulePath)
		if err != nil {
			continue
		}

		modules = append(modules, *module)
	}

	return modules, nil
}

func FindModule(codeDir, name string) (*Module, error) {
	modulePath := filepath.Join(codeDir, name)

	if _, err := os.Stat(modulePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("module %s not found", name)
	}

	return analyzeModule(name, modulePath)
}

func analyzeModule(name, modulePath string) (*Module, error) {
	manifestPath := detectManifestPath(modulePath)
	if manifestPath == "" {
		return nil, fmt.Errorf("no MANIFEST.MF found for %s", name)
	}

	needsSharedLib := detectSharedLibDependency(modulePath)

	return &Module{
		Name:           name,
		Path:           modulePath,
		ManifestPath:   manifestPath,
		NeedsSharedLib: needsSharedLib,
	}, nil
}

func detectManifestPath(modulePath string) string {
	rootManifest := filepath.Join(modulePath, "META-INF", "MANIFEST.MF")
	if _, err := os.Stat(rootManifest); err == nil {
		return "META-INF/MANIFEST.MF"
	}

	srcManifest := filepath.Join(modulePath, "src", "META-INF", "MANIFEST.MF")
	if _, err := os.Stat(srcManifest); err == nil {
		return "src/META-INF/MANIFEST.MF"
	}

	return ""
}

func detectSharedLibDependency(modulePath string) bool {
	srcDir := filepath.Join(modulePath, "src")
	found := false

	filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || found {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(path, ".java") {
			content, err := os.ReadFile(path)
			if err != nil {
				return nil
			}

			if strings.Contains(string(content), "import com.turning_leaf_technologies") {
				found = true
			}
		}
		return nil
	})

	return found
}

func isExcluded(name string, patterns []string) bool {
	for _, pattern := range patterns {
		if name == pattern {
			return true
		}
	}
	return false
}
