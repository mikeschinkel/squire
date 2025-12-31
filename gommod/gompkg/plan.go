package gompkg

import (
	"bytes"
	"fmt"
	"log/slog"
	"sort"

	"github.com/mikeschinkel/go-cliutil"
	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/go-dt/dtx"
	"github.com/mikeschinkel/gomion/gommod/goutils"
)

// PlanResult contains the results of repository initialization
type PlanResult struct {
	Initialized int
	Skipped     int
	Errors      []error
}

type PlanArgs struct {
	RepoDirs []string
	Config   *Config
	Logger   *slog.Logger
	Writer   cliutil.Writer
}

func getRepoDirsToScan(args PlanArgs) (dirs []dt.DirPath, err error) {
	dirs, err = dt.ParseDirPaths(args.RepoDirs)
	if len(args.RepoDirs) == 0 {
		dirs = args.Config.ScanDirs
	}
	if len(dirs) == 0 {
		var home dt.DirPath
		home, err = dt.UserHomeDir()
		if err != nil {
			goto end
		}
		dirs = []dt.DirPath{home}
	}
end:
	return dirs, err
}

// Plan handles the plan command. Is assumes a single repo to plan for and uses
// scan_dirs from user config to decide where to scan for pontentially dependent
// repositories.
func Plan(startDir string, args PlanArgs) (result *PlanResult, err error) {
	var goModFiles []dt.Filepath
	var graph *goutils.ModuleGraph
	var repoDir dt.DirPath
	var repoDirsToScan []dt.DirPath
	var traverseResult *goutils.TraverseResult
	// Check for mutual exclusivity

	// Ensure we have a directory to start with
	if startDir == "" {
		startDir = "."
	}

	// Ensure it is valid, and normalized
	repoDir, err = dt.ParseDirPath(startDir)
	if err != nil {
		goto end
	}

	// Convert to proper clean asolute URL in the case of "." provided
	repoDir, err = repoDir.Clean().Abs()
	if err != nil {
		goto end
	}

	// Ensure our start directory is actually a repo directory repoDir will different
	// from startDir and still be a repo is the repo is a parent dir of startDir.
	repoDir, err = FindRepoRoot(repoDir)
	if err != nil {
		goto end
	}

	// Get the list of all repos to find go.mod files in, which should include the
	// dir in repo too.
	repoDirsToScan, err = getRepoDirsToScan(args)
	if err != nil {
		goto end
	}

	// Collect up all go.mod file names found in repo dirs to scan
	goModFiles, err = FindGoModFiles[dt.Filepath](FindGoModFilesArgs{
		DirPaths:       repoDirsToScan,
		Config:         args.Config,
		ContinueOnErr:  false,
		SilenceErrs:    false,
		SkipBehavior:   SkipUnmanaged,
		MatchBehavior:  dtx.CollectOnMatch,
		ParseEntryFunc: nil,
		Logger:         args.Logger,
		Writer:         args.Writer,
	})
	if err != nil {
		goto end
	}

	// Sort go.mod files for deterministic graph building
	sort.Slice(goModFiles, func(i, j int) bool {
		return string(goModFiles[i]) < string(goModFiles[j])
	})

	// Now build the graph of all GoMod files
	graph = goutils.NewGraph(repoDir, goModFiles, goutils.ModuleGraphArgs{
		Logger: args.Logger,
		Writer: args.Writer,
	})
	err = graph.Build()
	if err != nil {
		goto end
	}

	traverseResult, err = graph.Traverse()
	if err != nil {
		goto end
	}

	// Format and print the grouped output
	for repoDir, modules := range traverseResult.RepoModules.Iterator() {
		fmt.Printf("\nRepo: %s\n", repoDir)
		for _, modDir := range modules {
			fmt.Printf("  Module: %s\n", modDir)
		}
	}

	result = &PlanResult{}
end:
	return result, err
}

// loadPlanFile loads an plan file which is a simple text file containing list of
// diretories that contains go.mod files, one file per file.
func loadPlanFile(file string) (files []dt.Filepath, err error) {
	var fp dt.Filepath
	var exists bool
	var content []byte
	var lines [][]byte

	fp, err = dt.ParseFilepath(file)
	if err != nil {
		goto end
	}
	exists, err = fp.Exists()
	if err != nil {
		goto end
	}
	if !exists {
		err = NewErr(dt.ErrFileNotExists, "filepath", file)
	}
	content, err = fp.ReadFile()
	if err != nil {
		goto end
	}
	lines = bytes.Split(content, []byte("\n"))
	files = make([]dt.Filepath, len(lines))
	for i, line := range lines {
		files[i] = dt.Filepath(line)
	}
end:
	return files, err
}
