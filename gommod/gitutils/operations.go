package gitutils

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"os/exec"
	"strings"

	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/go-dt/dtx"
)

type Streamer struct {
	Stdout io.Writer
	Stderr io.Writer
}

func NewStreamer(stdout io.Writer, stderr io.Writer) *Streamer {
	return &Streamer{Stdout: stdout, Stderr: stderr}
}

// Status runs git status in the module directory
func (s Streamer) Status(moduleDir dt.DirPath) error {
	cmd := exec.Command("git", "status")
	cmd.Dir = string(moduleDir)
	cmd.Stdout = s.Stdout
	cmd.Stderr = s.Stderr
	return cmd.Run()
}

// StageModuleFiles stages only files belonging to the current module (excluding nested modules)
func (s Streamer) StageModuleFiles(moduleDir dt.DirPath) (err error) {
	var filesToStage []string
	var nestedModules []string
	var cmd *exec.Cmd
	var out []byte
	var scanner *bufio.Scanner
	var line string
	var filename string
	var isNested bool

	// Find nested Go modules (subdirectories with go.mod)
	nestedModules, err = FindNestedModules(moduleDir)
	if err != nil {
		goto end
	}

	// Debug output
	if len(nestedModules) > 0 && s.Stdout != nil {
		dtx.Fprintf(s.Stdout, "Debug: Found %d nested module(s): %v\n", len(nestedModules), nestedModules)
	}

	// Unstage everything first
	cmd = exec.Command("git", "restore", "--staged", ".")
	cmd.Dir = string(moduleDir)
	err = cmd.Run()
	if err != nil {
		// Ignore error if there's nothing staged
		err = nil
	}

	// Get all files with changes (modified, added, deleted, untracked)
	cmd = exec.Command("git", "status", "--porcelain")
	cmd.Dir = string(moduleDir)
	out, err = cmd.Output()
	if err != nil {
		goto end
	}

	// Parse git status --porcelain output
	scanner = bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		line = scanner.Text()
		if len(line) < 4 {
			continue
		}
		// Format: "XY filename" where X is staged status, Y is unstaged status
		filename = line[3:]

		// Check if this file is in a nested module
		isNested = false
		for _, nestedModule := range nestedModules {
			if strings.HasPrefix(filename, nestedModule+"/") {
				isNested = true
				break
			}
		}

		if !isNested {
			filesToStage = append(filesToStage, filename)
		}
	}

	// Debug output
	if s.Stdout != nil {
		dtx.Fprintf(s.Stdout, "Debug: Staging %d file(s) for current module\n", len(filesToStage))
	}

	// Stage the filtered files
	if len(filesToStage) > 0 {
		args := append([]string{"add", "--"}, filesToStage...)
		cmd = exec.Command("git", args...)
		cmd.Dir = string(moduleDir)
		err = cmd.Run()
		if err != nil {
			goto end
		}
	}

	if s.Stdout != nil {
		dtx.Fprintf(s.Stdout, "\nStaged %d file(s)\n", len(filesToStage))
	}

end:
	return err
}

// UnstageAll unstages all files in the module directory
func (s Streamer) UnstageAll(moduleDir dt.DirPath) (err error) {
	var cmd *exec.Cmd

	cmd = exec.Command("git", "restore", "--staged", ".")
	cmd.Dir = string(moduleDir)
	err = cmd.Run()
	if err != nil {
		goto end
	}

	if s.Stdout != nil {
		dtx.Fprintf(s.Stdout, "\nUnstaged all changes\n")
	}

end:
	return err
}

// Commit commits staged changes with the given message
func (s Streamer) Commit(moduleDir dt.DirPath, message string) (err error) {
	var cmd *exec.Cmd

	cmd = exec.Command("git", "commit", "-m", message)
	cmd.Dir = string(moduleDir)
	cmd.Stdout = s.Stdout
	cmd.Stderr = s.Stderr
	err = cmd.Run()

	return err
}

var ErrStdErrOutput = errors.New("stderr output occurred")

// StageModuleFiles stages only files belonging to the current module (excluding nested modules)
func StageModuleFiles(moduleDir dt.DirPath) (out string, err error) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err = NewStreamer(&stdout, &stderr).StageModuleFiles(moduleDir)
	if err != nil {
		goto end
	}
	out = stdout.String()
	if len(stderr.Bytes()) == 0 {
		goto end
	}
	err = NewErr(ErrStdErrOutput, "stderr", stderr.String(), err)
end:
	return out, err
}

// UnstageAll unstages all files in the module directory
func UnstageAll(moduleDir dt.DirPath) (out string, err error) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err = NewStreamer(&stdout, &stderr).UnstageAll(moduleDir)
	if err != nil {
		goto end
	}
	out = stdout.String()
	if len(stderr.Bytes()) == 0 {
		goto end
	}
	err = NewErr(ErrStdErrOutput, "stderr", stderr.String(), err)
end:
	return out, err
}

// Commit commits staged changes with the given message
func Commit(moduleDir dt.DirPath, message string) (out string, err error) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err = NewStreamer(&stdout, &stderr).Commit(moduleDir, message)
	if err != nil {
		goto end
	}
	out = stdout.String()
	if len(stderr.Bytes()) == 0 {
		goto end
	}
	err = NewErr(ErrStdErrOutput, "stderr", stderr.String(), err)
end:
	return out, err
}

// FindNestedModules finds all Go modules in subdirectories (directories with go.mod)
func FindNestedModules(moduleDir dt.DirPath) (nestedModules []string, err error) {
	var cmd *exec.Cmd
	var out []byte
	var lines []string

	// Find all go.mod files in subdirectories (not the root)
	cmd = exec.Command("git", "ls-files", "*/go.mod")
	cmd.Dir = string(moduleDir)
	out, err = cmd.Output()
	if err != nil {
		goto end
	}

	lines = strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		// Extract directory path from "path/to/module/go.mod"
		dir := strings.TrimSuffix(line, "/go.mod")
		if dir != "" && dir != "go.mod" {
			nestedModules = append(nestedModules, dir)
		}
	}

end:
	return nestedModules, err
}

// StageFiles stages the specified files or directories in git
func StageFiles(moduleDir dt.DirPath, paths ...string) (err error) {
	var cmd *exec.Cmd
	var args []string

	if len(paths) == 0 {
		goto end
	}

	// Build git add command with all paths
	args = append([]string{"add"}, paths...)
	cmd = exec.Command("git", args...)
	cmd.Dir = string(moduleDir)
	err = cmd.Run()

end:
	return err
}
