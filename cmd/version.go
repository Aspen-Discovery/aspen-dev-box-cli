package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

var version = "dev"

func init() {
	rootCmd.Version = version
	rootCmd.SetVersionTemplate("adb {{.Version}}\n")
	rootCmd.AddCommand(VersionCommand())
}

func VersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:         "version",
		Short:       "Print the adb build version",
		Annotations: map[string]string{skipConfigValidation: "true"},
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("adb %s (%s %s/%s)\n", version, runtime.Version(), runtime.GOOS, runtime.GOARCH)
		},
	}
}
