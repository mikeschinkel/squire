package squiresvc

import (
	"fmt"

	"github.com/mikeschinkel/go-cfgstore"
	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/squire/squirepkg/gitutils"
	"github.com/mikeschinkel/squire/squirepkg/squire"
)

const (
	ArchivePath              dt.PathSegment = ".archive"
	JSONStatePersistenceFile dt.Filename    = "squire.json"
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

// InitSquireDirectoryArgs contains arguments for InitSquireDirectory
type InitSquireDirectoryArgs struct {
	ModuleDir     dt.DirPath
	CommitChanges bool // If true, commits the .squire/ structure to git
}

// InitSquireDirectory initializes the .squire/ directory structure
// Creates necessary subdirectories and configures git
func InitSquireDirectory(args InitSquireDirectoryArgs) (err error) {
	var squireDir dt.DirPath
	var files []dt.RelFilepath
	var needsArchive bool
	var needsExclude bool
	var ignoreFile *gitutils.IgnoreFile
	var excludeFile *gitutils.ExcludeFile
	var dirsProvider *cfgstore.DirsProvider

	// Get .squire directory using Custom config dir (relative to module, not repo root)
	dirsProvider = cfgstore.DefaultDirsProviderWithArgs(cfgstore.DirsProviderArgs{
		CustomDirPath: args.ModuleDir,
	})

	squireDir, err = cfgstore.ConfigDir(cfgstore.CustomConfigDirType, squire.ConfigSlug, dirsProvider)
	if err != nil {
		goto end
	}

	// Create main .squire directory
	err = squireDir.MkdirAll(0755)
	if err != nil {
		err = fmt.Errorf("failed to create .squire directory: %w", err)
		goto end
	}

	err = squireDir.MkSubdirs(ProjectConfigSubdirs, 0755)
	if err != nil {
		goto end
	}

	files = gitutils.KeepFiles(ProjectConfigSubdirs)
	err = squireDir.TouchFiles(files, 0644)
	if err != nil {
		goto end
	}

	// Create .archive subdirectories
	err = squireDir.Join(ArchivePath).MkSubdirs(ArchiveSubdirs, 0755)
	if err != nil {
		goto end
	}

	ignoreFile = gitutils.NewIgnoreFile(args.ModuleDir)
	// Check if .archive/ is already in .gitignore
	needsArchive, err = ignoreFile.ContainsPathSegment(ArchivePath)
	if err != nil {
		goto end
	}

	// Adds .squire/.archive/ to .gitignore, if not already present
	if !needsArchive {
		err = ignoreFile.AppendPathSegment(ArchivePath)
	}
	if err != nil {
		goto end
	}

	// Check if squire.json is already in .git/info/exclude
	excludeFile = gitutils.NewExcludeFile(args.ModuleDir)
	needsExclude, err = excludeFile.ContainsFilename(JSONStatePersistenceFile)
	if err != nil {
		goto end
	}
	// Adds squire.json to .git/info/exclude, if not already present (repo-specific, not in .gitignore)
	if !needsExclude {
		err = excludeFile.AppendFilename(JSONStatePersistenceFile)
	}
	if err != nil {
		goto end
	}

	// Commit the .squire/ structure if requested
	if args.CommitChanges {
		err = commitSquireStructure(args.ModuleDir, squireDir)
	}

end:
	return err
}

// commitSquireStructure stages and commits the .squire/ directory structure
func commitSquireStructure(moduleDir, squireDir dt.DirPath) (err error) {
	var repo *gitutils.Repo
	var hasChanges bool
	var squireRelPath dt.RelFilepath
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

	// Compute relative path from module dir to .squire
	// Use base name since .squire is always directly under moduleDir
	squireRelPath = dt.RelFilepath(squireDir.Base())
	filesToStage = append(filesToStage, squireRelPath)

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
	_, err = gitutils.Commit(moduleDir, "Initialize .squire/ directory structure\n\nCreated by Squire for managing commit workflow state.")

end:
	return err
}
