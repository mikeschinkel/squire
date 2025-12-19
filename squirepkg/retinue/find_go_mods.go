package retinue

import (
	"errors"
	"log/slog"
	"regexp"

	"github.com/mikeschinkel/go-cliutil"
	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/go-dt/dtx"
	"github.com/mikeschinkel/squire/squirepkg/squire"
)

func skipPaths() []dt.PathSegment {
	return []dt.PathSegment{
		// Version control
		".git",
		".hg",
		".svn",
		".bzr",

		// Dependency / package managers
		"node_modules",
		"vendor",

		// Go conventions
		"testdata",

		// Editors / IDEs
		".idea",
		".vscode",
		".vscode-test",

		// Build artifacts
		"bin",
		"dist",
		"build",
		"out",
		"obj",
		"target",

		// Infra / tooling caches
		".terraform",
		".terragrunt-cache",
		".helm",
		".kube",

		// OS junk (directories)
		".AppleDouble",
		".Spotlight-V100",
		".Trashes",
		"$RECYCLE.BIN",
		"System Volume Information",

		// Squire itself
		squire.ProjectConfigPath,
	}
}

type SkipBehavior int

const (
	SkipNone      SkipBehavior = 0
	SkipManaged   SkipBehavior = 1
	SkipUnmanaged SkipBehavior = 2
)

var goModRegexp = regexp.MustCompile(`go\.mod$`)
var pkgModRegexp = regexp.MustCompile(`pkg[/\\]mod$`)

type FindGoModFilesArgs struct {
	DirPaths       []dt.DirPath
	Config         *Config
	ContinueOnErr  bool
	SilenceErrs    bool
	SkipBehavior   SkipBehavior
	MatchBehavior  dtx.MatchBehavior
	ParseEntryFunc dtx.ParseEntryFunc
	Logger         *slog.Logger
	Writer         cliutil.Writer
}

type findGoModResultType interface {
	dt.Filepath | dt.RelFilepath | dt.DirPath | dt.EntryPath | dt.PathSegments
}

func FindGoModFiles[P findGoModResultType](args FindGoModFilesArgs) (files []P, err error) {
	var errs []error

	if args.DirPaths == nil {
		args.DirPaths = args.Config.ScanDirs
	}
	for _, dp := range args.DirPaths {
		var results []P
		results, err = findGoModFiles[P](dp, args)
		if err != nil {
			errs = append(errs, err)
		}
		files = append(files, results...)
	}
	if len(files) == 0 {
		args.Writer.Printf("No go.mod files found in %s\n", dtx.DirPaths(args.DirPaths).Join(", "))
		goto end
	}
	err = CombineErrs(errs)
end:
	return files, err
}

func findGoModFiles[P findGoModResultType](dp dt.DirPath, args FindGoModFilesArgs) (paths []P, err error) {
	repoCache := make(map[dt.DirPath]bool)
	errs := make([]error, 0)
	skipdirFunc := func(root dt.DirPath, de *dt.DirEntry) (skip bool) {
		skip = true
		if pkgModRegexp.MatchString(string(de.Rel)) {
			// Special-case skipping Go module cache: ${GOPATH}/pkg/mod
			de.SkipDir()
		}
		return skip
	}
	skipEntryFunc := func(root dt.DirPath, de *dt.DirEntry) (skip bool) {
		return maybeSkipEntry(root, de, repoCache, &errs, args)
	}
	scanner := dtx.NewDirPathScanner(dp, dtx.DirPathScannerArgs{
		ContinueOnErr:  args.ContinueOnErr,
		Writer:         args.Writer,
		MatchBehavior:  args.MatchBehavior,
		SkipPaths:      skipPaths(),
		SkipDirFunc:    skipdirFunc,
		SkipEntryFunc:  skipEntryFunc,
		ParseEntryFunc: args.ParseEntryFunc,
	})
	var entries []dt.EntryPath
	entries, err = scanner.Scan()
	errs = AppendErr(errs, err)
	err = CombineErrs(errs)
	paths = make([]P, len(entries))
	for i, e := range entries {
		paths[i] = P(e)
	}
	return paths, err
}

func maybeSkipEntry(root dt.DirPath, de *dt.DirEntry, managedCache map[dt.DirPath]bool, errs *[]error, args FindGoModFilesArgs) (skip bool) {
	var moduleDir dt.DirPath
	var repoRoot dt.DirPath
	var ok bool
	var path dt.Filepath
	var err error

	if !goModRegexp.MatchString(string(de.Rel)) {
		skip = true
		goto end
	}

	// Build absolute filepath from root and relative path
	path = dt.FilepathJoin(root, de.Rel)

	// Get module directory
	moduleDir = path.Dir()

	// Find repo root
	repoRoot, err = FindRepoRoot(moduleDir)
	switch {
	case errors.Is(err, ErrRepoRootNotFound):
		err = nil
		goto end
	case err != nil:
		err = NewErr(ErrFindRepoError, "module_dir", moduleDir, err)
		*errs = append(*errs, err)
		if !args.SilenceErrs {
			// Log errors but continue scanning
			args.Logger.Warn(ErrNoRepoRoot.Error(), "path", path, "error", err)
		}
		err = nil
		goto end
	default:
		// Carry on
	}

	// Check managedCache first
	_, ok = managedCache[repoRoot]
	if !ok {
		var managed bool
		managed, err = isRepoManaged(repoRoot)
		if err != nil {
			goto end
		}
		managedCache[repoRoot] = managed
	}

	if !managedCache[repoRoot] {
		goto end
	}
	skip = true
	switch {
	case args.SkipBehavior == SkipManaged && managedCache[repoRoot]:

	case args.SkipBehavior == SkipUnmanaged && !managedCache[repoRoot]:
	case args.SkipBehavior == SkipNone:
		fallthrough
	default:
		skip = false
	}

end:
	if skip && de.IsDir() {
		de.SkipDir()
	}
	return skip
}
