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

// InitSquireDirectory initializes the .squire/ directory structure
// Creates necessary subdirectories and configures git
func InitSquireDirectory(modDir dt.DirPath) (err error) {
	var squireDir dt.DirPath
	var files []dt.RelFilepath
	var needsArchive bool
	var needsExclude bool
	var ignoreFile *gitutils.IgnoreFile
	var excludeFile *gitutils.ExcludeFile

	squireDir, err = cfgstore.ProjectConfigDir(squire.ConfigSlug, nil)
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

	ignoreFile = gitutils.NewIgnoreFile(modDir)
	// Check if .archive/ is already in .gitignore
	needsArchive, err = ignoreFile.ContainsPathSegment(ArchivePath)
	if err != nil {
		goto end
	}

	// Adds .squire/.archive/ to .gitignore, if needed
	if needsArchive {
		err = ignoreFile.AppendPathSegment(ArchivePath)
	}
	if err != nil {
		goto end
	}

	// Check if .git/info/exclude is already in .gitignore
	excludeFile = gitutils.NewExcludeFile(modDir)
	needsExclude, err = excludeFile.ContainsFilename(JSONStatePersistenceFile)
	if err != nil {
		goto end
	}
	// Adds squire.json to .git/info/exclude, if needed (repo-specific, not in .gitignore)
	if needsExclude {
		err = excludeFile.AppendFilename(JSONStatePersistenceFile)
	}
	if err != nil {
		goto end
	}

end:
	return err
}
