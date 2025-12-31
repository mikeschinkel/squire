package gompkg

import (
	"fmt"

	"github.com/mikeschinkel/go-cfgstore"
	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/gomion/gommod/gitutils"
	"github.com/mikeschinkel/gomion/gommod/gomion"
)

const (
	ArchivePath              dt.PathSegment = ".archive"
	JSONStatePersistenceFile dt.Filename    = "gomion.json"
)

var ProjectConfigSubdirs = []dt.PathSegments{
	"plans",      // Staging plans
	"candidates", // Commit candidates
	"snapshots",  // Staging snapshots
}
var ArchiveSubdirs = []dt.PathSegments{
	"candidates", // Archived candidates
	"snapshots",  // Archived snapshots
}

// InitGomionDirectoryArgs contains arguments for InitGomionDirectory
type InitGomionDirectoryArgs struct {
	ModuleDir     dt.DirPath
	CommitChanges bool // If true, commits the .gomion/ structure to git
}

// InitGomionDirectory initializes the .gomion/ directory structure
// Creates necessary subdirectories and configures git
func InitGomionDirectory(args InitGomionDirectoryArgs) (err error) {
	var gomionDir dt.DirPath
	var files []dt.RelFilepath
	var needsArchive bool
	var needsExclude bool
	var ignoreFile *gitutils.IgnoreFile
	var excludeFile *gitutils.ExcludeFile
	var dirsProvider *cfgstore.DirsProvider

	// Get .gomion directory using Custom config dir (relative to module, not repo root)
	dirsProvider = cfgstore.DefaultDirsProviderWithArgs(cfgstore.DirsProviderArgs{
		CustomDirPath: args.ModuleDir,
	})

	gomionDir, err = cfgstore.ConfigDir(cfgstore.CustomConfigDirType, gomion.ConfigSlug, dirsProvider)
	if err != nil {
		goto end
	}

	// Create main .gomion directory
	err = gomionDir.MkdirAll(0755)
	if err != nil {
		err = fmt.Errorf("failed to create .gomion directory: %w", err)
		goto end
	}

	err = gomionDir.MkSubdirs(ProjectConfigSubdirs, 0755)
	if err != nil {
		goto end
	}

	files = gitutils.KeepFiles(ProjectConfigSubdirs)
	err = gomionDir.TouchFiles(files, 0644)
	if err != nil {
		goto end
	}

	// Create .archive subdirectories
	err = gomionDir.Join(ArchivePath).MkSubdirs(ArchiveSubdirs, 0755)
	if err != nil {
		goto end
	}

	ignoreFile = gitutils.NewIgnoreFile(args.ModuleDir)
	// Check if .archive/ is already in .gitignore
	needsArchive, err = ignoreFile.ContainsPathSegment(ArchivePath)
	if err != nil {
		goto end
	}

	// Adds .gomion/.archive/ to .gitignore, if not already present
	if !needsArchive {
		err = ignoreFile.AppendPathSegment(ArchivePath)
	}
	if err != nil {
		goto end
	}

	// Check if gomion.json is already in .git/info/exclude
	excludeFile = gitutils.NewExcludeFile(args.ModuleDir)
	needsExclude, err = excludeFile.ContainsFilename(JSONStatePersistenceFile)
	if err != nil {
		goto end
	}
	// Adds gomion.json to .git/info/exclude, if not already present (repo-specific, not in .gitignore)
	if !needsExclude {
		err = excludeFile.AppendFilename(JSONStatePersistenceFile)
	}
	if err != nil {
		goto end
	}

	// Commit the .gomion/ structure if requested
	if args.CommitChanges {
		err = commitGomionStructure(args.ModuleDir, gomionDir)
	}

end:
	return err
}

// commitGomionStructure stages and commits the .gomion/ directory structure
func commitGomionStructure(moduleDir, gomionDir dt.DirPath) (err error) {
	var repo *gitutils.Repo
	var hasChanges bool
	var gomionRelPath dt.RelFilepath
	var gitignorePath dt.Filepath
	var gitignoreExists bool
	var filesToStage []dt.RelFilepath
	var pathStrings []string

	// Open repo
	repo, err = gitutils.Open(moduleDir)
	if err != nil {
		goto end
	}

	// Check if there are any changes to commit
	hasChanges, err = repo.IsDirty()
	if err != nil {
		goto end
	}

	if !hasChanges {
		// No changes to commit
		goto end
	}

	// Compute relative path from module dir to .gomion
	// Use base name since .gomion is always directly under moduleDir
	gomionRelPath = dt.RelFilepath(gomionDir.Base())
	filesToStage = append(filesToStage, gomionRelPath)

	// Add .gitignore if it exists
	gitignorePath = dt.FilepathJoin(moduleDir, gitutils.IgnoreFilename)
	gitignoreExists, err = gitignorePath.Exists()
	if err != nil {
		goto end
	}

	if gitignoreExists {
		filesToStage = append(filesToStage, dt.RelFilepath(gitutils.IgnoreFilename))
	}

	// Convert dt types to strings for git boundary
	pathStrings = make([]string, len(filesToStage))
	for i, p := range filesToStage {
		pathStrings[i] = string(p)
	}

	// Stage the files
	err = gitutils.StageFiles(moduleDir, pathStrings...)
	if err != nil {
		goto end
	}

	// Commit the changes
	_, err = gitutils.Commit(moduleDir, "Initialize .gomion/ directory structure\n\nCreated by Gomion for managing commit workflow state.")

end:
	return err
}
