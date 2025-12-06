package squirecmds

import (
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
	dirArg string
	roots  []dt.DirPath
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
	var allGoMods []dt.Filepath
	var root dt.DirPath
	var goMods []dt.Filepath
	var goModPath dt.Filepath

	config = c.Config.(*squire.Config)

	// Parse directory argument (default to current directory if none provided)
	c.roots, err = c.parseDirectoryArgs()
	if err != nil {
		goto end
	}

	// Scan each root directory
	for _, root = range c.roots {
		goMods, err = c.scanDirectory(root, config)
		if err != nil {
			goto end
		}
		allGoMods = append(allGoMods, goMods...)
	}

	// Output all discovered go.mod paths, one per line
	for _, goModPath = range allGoMods {
		config.Writer.Printf("%s\n", goModPath)
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

// scanDirectory recursively scans a directory for go.mod files
func (c *ScanCmd) scanDirectory(root dt.DirPath, config *squire.Config) (goMods []dt.Filepath, err error) {
	var repoCache map[string]bool
	var walkFunc func(path string, info os.FileInfo, walkErr error) error

	repoCache = make(map[string]bool)

	walkFunc = func(path string, info os.FileInfo, walkErr error) error {
		var entryPath dt.EntryPath
		var goModPath dt.Filepath
		var moduleDir dt.DirPath
		var repoRoot dt.DirPath
		var isManaged bool
		var cached bool
		var repoRootStr string

		if walkErr != nil {
			config.Logger.Warn("error accessing path", "path", path, "error", walkErr)
			goto end
		}

		// Skip if not a go.mod file
		if info.IsDir() || info.Name() != "go.mod" {
			goto end
		}

		// Convert to dt types
		entryPath = dt.EntryPath(path)
		goModPath = dt.Filepath(entryPath)

		// Get module directory
		moduleDir = goModPath.Dir()

		// Find repo root
		repoRoot, err = findRepoRoot(moduleDir)
		if err != nil {
			config.Logger.Warn("could not find repo root", "path", path, "error", err)
			err = nil // Continue scanning
			goto end
		}

		// Check cache first
		repoRootStr = string(repoRoot)
		isManaged, cached = repoCache[repoRootStr]
		if !cached {
			isManaged = isRepoManaged(repoRoot)
			repoCache[repoRootStr] = isManaged
		}

		// Only include if repo is not managed
		if !isManaged {
			goMods = append(goMods, goModPath)
		}

	end:
		return err
	}

	err = filepath.Walk(string(root), walkFunc)
	if err != nil {
		err = NewErr(ErrCommand, ErrScan, ErrFileOperation, "root", root, err)
		goto end
	}

end:
	return goMods, err
}

// findRepoRoot walks up the directory tree to find the nearest .git directory
func findRepoRoot(startDir dt.DirPath) (repoRoot dt.DirPath, err error) {
	var dir dt.DirPath
	var gitPath dt.DirPath
	var exists bool
	var parent dt.DirPath

	dir = startDir

	for {
		gitPath = dt.DirPathJoin(dir, ".git")
		exists, err = gitPath.Exists()
		if err != nil {
			err = NewErr(ErrRepoRoot, "path", gitPath, err)
			goto end
		}

		if exists {
			repoRoot = dir
			goto end
		}

		parent = dir.Dir()
		if parent == dir {
			// Reached filesystem root without finding .git
			err = NewErr(ErrRepoRoot, dt.ErrFileDoesNotExist, "start_dir", startDir)
			goto end
		}
		dir = parent
	}

end:
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
