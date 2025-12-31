package gitutils

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/mikeschinkel/go-dt"
)

// GetStagedFiles returns the list of files currently staged in the index
func (r *Repo) GetStagedFiles(ctx context.Context) (files []dt.RelFilepath, err error) {
	var out string
	var lines []string

	// Get list of staged files using --cached --name-only
	out, err = r.runGit(ctx, r.Root, "diff", "--cached", "--name-only")
	if err != nil {
		goto end
	}

	lines = strings.Split(strings.TrimSpace(out), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		files = append(files, dt.RelFilepath(line))
	}

end:
	return files, err
}

// ExportStagedArgs contains arguments for ExportStagedFiles
type ExportStagedArgs struct {
	Repo         *Repo
	DestDir      dt.DirPath
	ModuleRelDir dt.PathSegments // Optional: filter to files within this module directory
}

// ExportStagedFiles exports staged file contents to a destination directory
// using `git show :path` to get the staged version of each file.
//
// This is critical for pre-commit analysis: we need to analyze what WILL be
// committed (staged version) not what's in the working directory (which may
// have unstaged changes).
//
// Edge cases handled:
// - Deleted files: git show fails, we skip them (expected)
// - New files: works correctly (they're in index)
// - Renamed files: appears as delete + add (both handled correctly)
// - Partial staging: returns staged version only (correct)
// - Binary files: copied as-is
func ExportStagedFiles(ctx context.Context, args ExportStagedArgs) (err error) {
	var files []dt.RelFilepath
	var modulePrefix string

	if args.Repo == nil {
		err = fmt.Errorf("repo cannot be nil")
		goto end
	}

	if args.DestDir == "" {
		err = fmt.Errorf("destination directory cannot be empty")
		goto end
	}

	// Get list of staged files
	files, err = args.Repo.GetStagedFiles(ctx)
	if err != nil {
		goto end
	}

	// Set up module path prefix for filtering if specified
	if args.ModuleRelDir != "" && args.ModuleRelDir != "." {
		modulePrefix = string(args.ModuleRelDir) + "/"
	}

	// Export each staged file
	for _, file := range files {
		var shouldExport bool
		var fileStr string
		var destPath dt.Filepath
		var content []byte

		fileStr = string(file)

		// Filter to module directory if specified
		if modulePrefix != "" {
			shouldExport = strings.HasPrefix(fileStr, modulePrefix)
			if !shouldExport {
				continue
			}
		}

		// Use `git show :path` to get staged version
		// The :path syntax means "the version in the index"
		content, err = args.Repo.gitShowStaged(ctx, fileStr)
		if err != nil {
			// File might be deleted (staged for deletion)
			// This is expected - skip it and continue
			err = nil
			continue
		}

		// Build destination path preserving directory structure
		destPath = dt.FilepathJoin(args.DestDir, dt.RelFilepath(fileStr))

		// Create parent directory if it doesn't exist
		err = destPath.Dir().MkdirAll(0o755)
		if err != nil {
			goto end
		}

		// Write file content
		err = os.WriteFile(string(destPath), content, 0o644)
		if err != nil {
			goto end
		}
	}

end:
	return err
}

// gitShowStaged uses `git show :path` to get the staged version of a file
func (r *Repo) gitShowStaged(ctx context.Context, relPath string) (content []byte, err error) {
	var cmd = fmt.Sprintf(":%s", relPath)
	var out string

	out, err = r.runGit(ctx, r.Root, "show", cmd)
	if err != nil {
		goto end
	}

	content = []byte(out)

end:
	return content, err
}

// GetStagedDiff returns the diff of staged changes
func (r *Repo) GetStagedDiff(ctx context.Context) (diff string, err error) {
	diff, err = r.runGit(ctx, r.Root, "diff", "--cached")
	return diff, err
}

// FindBaselineTag finds the most recent reachable semver tag to use as a baseline
// for comparison. This is typically the latest release tag.
func (r *Repo) FindBaselineTag(ctx context.Context, moduleRelDir dt.PathSegments) (tag string, err error) {
	var headCommit string

	// Get current HEAD commit
	headCommit, err = r.RevParse("HEAD")
	if err != nil {
		goto end
	}

	// Find latest reachable semver tag
	tag, err = r.LatestTag(ctx, headCommit, &LatestTagArgs{
		ModuleRelPath: dt.RelDirPath(moduleRelDir),
	})

end:
	return tag, err
}
