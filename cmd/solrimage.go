package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

var luceneMatchVersionPattern = regexp.MustCompile(`<luceneMatchVersion>(\d+)`)

func setupSolrImage(cloneDir string) {
	configPaths, err := filepath.Glob(filepath.Join(cloneDir, "data_dir_setup", "solr*", "*", "conf", "solrconfig.xml"))
	if err != nil || len(configPaths) == 0 {
		return
	}
	if os.Getenv("SOLR_CONFIGSETS") == "" {
		os.Setenv("SOLR_CONFIGSETS", filepath.Dir(filepath.Dir(filepath.Dir(configPaths[0]))))
	}
	if os.Getenv("SOLR_IMAGE") != "" {
		return
	}
	content, err := os.ReadFile(configPaths[0])
	if err != nil {
		return
	}
	match := luceneMatchVersionPattern.FindSubmatch(content)
	if match == nil {
		return
	}
	major := string(match[1])
	image := "solr:" + major
	os.Setenv("SOLR_IMAGE", image)
	fmt.Printf("Solr %s configs detected in %s — using image %s\n", major, cloneDir, image)
}
