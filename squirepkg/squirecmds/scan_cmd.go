package squirecmds

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/mikeschinkel/go-cliutil"
	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/squire/squirepkg/squire"
)

var _ cliutil.CommandHandler = (*ScanCmd)(nil)

// ScanCmd discovers Go modules under specified roots
type ScanCmd struct {
	*cliutil.CmdBase
	dirArg        string
	continueOnErr bool
	roots         []dt.DirPath
}

func init() {
	var err error
	var cmd *ScanCmd

	cmd = &ScanCmd{
		dirArg: "",
	}

	cmd.CmdBase = cliutil.NewCmdBase(cliutil.CmdArgs{
		Order:       10,
		Name:        "scan",
		Usage:       "scan [<dir>]",
		Description: "Discover Go modules in unmanaged repos (defaults to current directory)",
		FlagDefs: []cliutil.FlagDef{
			{
				Name:     "continue",
				Usage:    "On error, accumulate and continue; do not fail fast",
				Required: false,
				Bool:     &cmd.continueOnErr,
			},
		},
		ArgDefs: []*cliutil.ArgDef{
			{
				Name:     "dir",
				Usage:    "Directory to scan (defaults to current directory)",
				Required: false,
				String:   &cmd.dirArg,
				Example:  "~/Projects",
			},
		},
	})

	err = cliutil.RegisterCommand(cmd)
	if err != nil {
		panic(err)
	}
}

// Handle executes the scan command
func (c *ScanCmd) Handle() (err error) {
	var config *squire.Config
	var root dt.DirPath
	var errs []error

	config = c.Config.(*squire.Config)

	// Parse directory argument (default to current directory if none provided)
	c.roots, err = c.parseDirectoryArgs()
	if err != nil {
		goto end
	}

	// Scan each root directory
	for _, root = range c.roots {
		err = c.scanDirectory(root, config)
		if err != nil {
			if c.continueOnErr {
				errs = append(errs, err)
				continue
			}
			goto end
		}
	}

end:
	return err
}

// parseDirectoryArgs parses the directory argument (defaults to current directory if none provided)
func (c *ScanCmd) parseDirectoryArgs() (roots []dt.DirPath, err error) {
	var arg string
	var root dt.DirPath

	// Default to current directory if no arg provided
	if c.dirArg == "" {
		arg = "."
	} else {
		arg = c.dirArg
	}

	// Parse and expand the directory path
	root, err = parseAndExpandDirPath(arg)
	if err != nil {
		goto end
	}

	roots = []dt.DirPath{root}

end:
	return roots, err
}

var ErrAccessingPath = errors.New("error accessing path")
var ErrNoRepoRoot = errors.New("no repository root")

// scanDirectory recursively scans a directory for go.mod files
func (c *ScanCmd) scanDirectory(root dt.DirPath, config *squire.Config) (err error) {
	var repoCache map[string]bool
	var de dt.DirEntry
	var goModPath dt.Filepath
	var moduleDir dt.DirPath
	var repoRoot dt.DirPath
	var isManaged bool
	var cached bool
	var repoRootStr string

	var errs []error

	repoCache = make(map[string]bool)

	for de, err = range root.Walk() {
		if err != nil {
			config.Logger.Warn(ErrAccessingPath.Error(), "path", de.Rel, "error", err)
			err = NewErr(ErrAccessingPath, "path", de.Rel, err)
			if c.continueOnErr {
				errs = append(errs, err)
				err = nil
				continue
			}
			goto end
		}

		// Skip if not a go.mod file
		if de.IsDir() || de.Entry.Name() != "go.mod" {
			continue
		}

		// Build absolute filepath from root and relative path
		goModPath = dt.FilepathJoin(root, de.Rel)

		// Get module directory
		moduleDir = goModPath.Dir()

		// Find repo root
		repoRoot, err = findRepoRoot(moduleDir)
		switch {
		case errors.Is(err, dt.ErrDoesNotExist):
			err = nil
			continue
		case err != nil:
			config.Logger.Warn(ErrNoRepoRoot.Error(), "path", goModPath, "error", err)
			err = NewErr(ErrNoRepoRoot, "path", goModPath, err)
			if c.continueOnErr {
				errs = append(errs, err)
				err = nil
				continue
			}
			goto end
		default:
			// Carry on
		}

		// Check cache first
		repoRootStr = string(repoRoot)
		isManaged, cached = repoCache[repoRootStr]
		if !cached {
			isManaged = isRepoManaged(repoRoot)
			repoCache[repoRootStr] = isManaged
		}

		// Only include if repo is not managed
		if isManaged {
			continue
		}
		c.Writer.Printf("%s\n", goModPath)
	}
	err = CombineErrs(errs)
end:
	return err
}

// findRepoRoot walks up the directory tree to find the nearest .git directory
func findRepoRoot(startDir dt.DirPath) (repoRoot dt.DirPath, err error) {
	var dir dt.DirPath
	var gitDir dt.DirPath
	var exists bool
	var parent dt.DirPath

	dir = startDir

	for {
		gitDir = dt.DirPathJoin(dir, ".git")
		exists, err = gitDir.Exists()
		if err != nil {
			if !os.IsNotExist(err) {
				err = WithErr(err, dt.ErrFileSystem)
				goto end
			}
			err = NewErr(dt.ErrDirDoesNotExist, dt.ErrDoesNotExist, "path", gitDir, err)
			goto end
		}

		if exists {
			repoRoot = dir
			goto end
		}

		parent = dir.Dir()
		if parent == dir {
			// Reached filesystem root without finding .git
			err = NewErr(dt.ErrFileDoesNotExist, dt.ErrDoesNotExist, "start_dir", startDir)
			goto end
		}
		dir = parent
	}

end:
	if err != nil {
		err = WithErr(err, ErrRepoRoot)
	}
	return repoRoot, err
}

// isRepoManaged checks if a repo has a .squire/config.json file
func isRepoManaged(repoRoot dt.DirPath) (managed bool) {
	var configPath dt.Filepath
	var err error

	squireDir := dt.DirPathJoin(repoRoot, ".squire")
	configPath = dt.FilepathJoin(squireDir, "config.json")
	managed, err = configPath.Exists()
	if err != nil {
		// If we can't determine, assume not managed
		managed = false
	}

	return managed
}

// parseAndExpandDirPath expands ~ and converts to absolute DirPath
func parseAndExpandDirPath(pathStr string) (dirPath dt.DirPath, err error) {
	var expanded dt.DirPath
	var absPath dt.DirPath
	var exists bool

	// Convert string to DirPath and expand tilde
	expanded, err = expandTildeDirPath(dt.DirPath(pathStr))
	if err != nil {
		err = NewErr(ErrFileOperation, "path", pathStr, err)
		goto end
	}

	// Make absolute
	absPath, err = absDirPath(expanded)
	if err != nil {
		err = NewErr(ErrFileOperation, "path", expanded, err)
		goto end
	}

	// Verify directory exists
	exists, err = absPath.Exists()
	if err != nil {
		err = NewErr(dt.ErrNotADirectory, "path", absPath, err)
		goto end
	}

	if !exists {
		err = NewErr(dt.ErrFileDoesNotExist, "path", absPath)
		goto end
	}

	dirPath = absPath

end:
	return dirPath, err
}

// expandTildeDirPath expands ~ to the user's home directory for DirPath
func expandTildeDirPath(path dt.DirPath) (expanded dt.DirPath, err error) {
	var pathStr string
	var home string
	var homeDir dt.DirPath

	pathStr = string(path)

	if !strings.HasPrefix(pathStr, "~") {
		expanded = path
		goto end
	}

	home, err = os.UserHomeDir()
	if err != nil {
		goto end
	}
	homeDir = dt.DirPath(home)

	if pathStr == "~" {
		expanded = homeDir
		goto end
	}

	if strings.HasPrefix(pathStr, "~/") {
		expanded = dt.DirPathJoin(homeDir, pathStr[2:])
		goto end
	}

	expanded = path

end:
	return expanded, err
}

// absDirPath returns absolute DirPath
func absDirPath(path dt.DirPath) (absPath dt.DirPath, err error) {
	var abs string

	abs, err = filepath.Abs(string(path))
	if err != nil {
		goto end
	}

	absPath = dt.DirPath(abs)

end:
	return absPath, err
}
