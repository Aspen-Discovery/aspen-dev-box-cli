package cmd

import (
    "fmt"
    "os"
    "os/exec"
    "strings"
    "path/filepath"

    "github.com/spf13/cobra"
)

func init() {
    rootCmd.AddCommand(JarBuildCommand())
}

func runDockerBuild(aspenClone, jarFile string) error {
    workDir := fmt.Sprintf("/app/code/%s", jarFile)

    fmt.Printf("\n\033[1;34mRecompiling JAR file: %s\033[0m\n", jarFile)

    // Use the command variable
    command := exec.Command("docker", "run", "--rm",
        "-v", fmt.Sprintf("%s:/app", aspenClone),
        "-w", workDir,
        "--user", fmt.Sprintf("%d:%d", os.Getuid(), os.Getgid()),
        "adoptopenjdk:11", "bash", "-c", `
            mkdir -p bin && \
            javac -cp "$(find /app -name '*.jar' | tr '\n' ':')" -d bin $(find src -name '*.java') $(find /app/code/java_shared_libraries -name '*.java') && \
            jar cfm $(basename $(pwd)).jar META-INF/MANIFEST.MF -C bin . && \
            rm -rf bin
        `,
    )

    // Run the command
    command.Stdin = os.Stdin
    command.Stdout = os.Stdout
    command.Stderr = os.Stderr

    return command.Run()
}

func JarBuildCommand() *cobra.Command {
    var all bool

    cmd := &cobra.Command{
        Use:   "jarbuild",
        Short: "Run the Jar builder command",
        Run: func(cmd *cobra.Command, args []string) {
            aspenClone := os.Getenv("ASPEN_CLONE")
            if aspenClone == "" {
                fmt.Println("Error: ASPEN_CLONE environment variable not set.")
                os.Exit(1)
            }

            if all {
                // Run the find command to get all JAR files
                findCmd := exec.Command("docker", "run", "--rm",
                    "-v", fmt.Sprintf("%s:/app", aspenClone),
                    "-w", "/app",
                    "alpine:latest", "sh", "-c", `
                        apk add --no-cache findutils > /dev/null && \
                        find /app/code -mindepth 2 -maxdepth 2 -name '*.jar' | grep -v "java_shared_libraries" | grep -v "marcMergeUtility" | grep -v "palace_project_export" | grep -v "rbdigital_export" | xargs -n 1 basename | sed 's/\.jar$//'
                    `,
                )

                findOutput, err := findCmd.Output()
                if err != nil {
                    fmt.Printf("Error finding JAR files: %v\n", err)
                    os.Exit(1)
                }

                jarFiles := strings.Split(strings.TrimSpace(string(findOutput)), "\n")
                for _, jarFile := range jarFiles {
                    if jarFile == "" {
                        continue
                    }

                    if err := runDockerBuild(aspenClone, jarFile); err != nil {
                        fmt.Printf("Error running Docker command for %s: %v\n", jarFile, err)
                        os.Exit(1)
                    }
                }
            } else {
                // Create a temporary file to capture the output of fzf
                tmpFile, err := os.CreateTemp(aspenClone, "fzf-output")
                if err != nil {
                    fmt.Printf("Error creating temporary file: %v\n", err)
                    os.Exit(1)
                }
                defer os.Remove(tmpFile.Name())

                // Get the temporary file name inside the Docker container
                tmpFileName := filepath.Join("/app", filepath.Base(tmpFile.Name()))

                fzfCmd := exec.Command("docker", "run", "--rm", "-it",
                    "-v", fmt.Sprintf("%s:/app", aspenClone),
                    "-w", "/app",
                    "alpine:latest", "sh", "-c", fmt.Sprintf(`
                        apk add --no-cache fzf findutils > /dev/null && \
                        find /app/code -mindepth 2 -maxdepth 2 -name '*.jar' | grep -v "java_shared_libraries" | grep -v "marcMergeUtility" | grep -v "palace_project_export" | grep -v "rbdigital_export" | xargs -n 1 basename | sed 's/\.jar$//' | fzf > %s
                    `, tmpFileName),
                )

                // Ensure the terminal input and output are correctly passed through
                fzfCmd.Stdin = os.Stdin
                fzfCmd.Stdout = os.Stdout
                fzfCmd.Stderr = os.Stderr

                if err := fzfCmd.Run(); err != nil {
                    fmt.Printf("Error selecting JAR file with fzf: %v\n", err)
                    os.Exit(1)
                }

                // Read the output from the temporary file
                fzfOutput, err := os.ReadFile(filepath.Join(aspenClone, filepath.Base(tmpFile.Name())))
                if err != nil {
                    fmt.Printf("Error reading fzf output: %v\n", err)
                    os.Exit(1)
                }

                selectedJar := strings.TrimSpace(string(fzfOutput))
                if selectedJar == "" {
                    fmt.Println("No JAR file selected.")
                    os.Exit(1)
                }

                if err := runDockerBuild(aspenClone, selectedJar); err != nil {
                    fmt.Printf("Error running Docker command: %v\n", err)
                    os.Exit(1)
                }
            }
        },
    }

    cmd.Flags().BoolVarP(&all, "all", "a", false, "Build all JAR files")
    return cmd
}
