package gompkg

import (
	"path"

	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/gomion/gommod/gomion"
)

// ModuleSpec is a type for Go modules that may '*' or '?' character for matching said modules
// Examples:
//
//	github.com/mikeschinkel/go-*",
//	 "github.com/xmlui-org/*"
type ModuleSpec string

// Matches checks if a module path matches a module spec pattern
// Patterns use simple glob syntax: github.com/mikeschinkel/go-*
func (ms ModuleSpec) Matches(module string) (matches bool) {
	var err error
	matches, err = path.Match(string(ms), module)
	if err != nil {
		// Invalid pattern - return false
		matches = false
		goto end
	}
end:
	return matches
}

type ModuleSpecs []ModuleSpec

func ParseModuleSpec(s string) (_ ModuleSpec, err error) {
	// TODO: Add more validation here
	return ModuleSpec(s), err
}

func ParseModuleSpecs(moduleSpecs []string) (mss []ModuleSpec, err error) {
	var errs []error
	mss = make([]ModuleSpec, len(moduleSpecs))
	for i, ms := range moduleSpecs {
		mss[i], err = ParseModuleSpec(ms)
		errs = gomion.AppendErr(errs, err)
	}
	err = gomion.CombineErrs(errs)
	if err != nil {
		// TODO: Add error sentinel
		mss = nil
	}
	return mss, err
}

// findGoModFiles scans a directory for repos whose modules match module specs
func (moduleSpecs ModuleSpecs) ScanDir(scanDir dt.DirPath, currentRepo dt.DirPath) (matches []dt.TildeDirPath, err error) {
	var de dt.DirEntry
	var goModPath dt.Filepath
	var moduleDir dt.DirPath
	var repoRoot dt.DirPath
	var modulePath string
	var spec ModuleSpec
	var errs []error
	var managed bool

	matches = make([]dt.TildeDirPath, 0)

	// Walk scan directory looking for go.mod files
	for de, err = range scanDir.Walk() {
		if err != nil {
			// TODO — ONLY skip permission errors, fail on others
			err = nil
			continue // Skip access errors
		}

		// Only process files
		if de.IsDir() {
			if unnecessaryDirsRegex.MatchString(string(de.Rel)) {
				// No need to drill down into .git directories
				de.SkipDir()
			}
			continue
		}

		// Only process go.mod files
		if de.Entry.Name() != "go.mod" {
			continue
		}

		// Get full path to go.mod
		goModPath = dt.FilepathJoin(scanDir, de.Rel)
		moduleDir = goModPath.Dir()

		// Find repo root
		repoRoot, err = FindRepoRoot(moduleDir)
		if err != nil {
			err = nil
			continue // Skip if not a git repo
		}

		// Skip if this is the current repo
		if repoRoot == currentRepo {
			continue
		}

		// Check if already managed (skip if has .gomion/config.json)

		managed, err = isRepoManaged(repoRoot)
		if managed {
			continue
		}

		// Parse go.mod to get module path
		modulePath, err = extractModulePath(goModPath)
		if err != nil {
			errs = AppendErr(errs, NewErr(ErrCannotExtractModulePath, "go_mod", goModPath, err))
			continue // Skip if can't parse
		}

		// Check if module path matches any module spec pattern
		for _, spec = range moduleSpecs {
			if !spec.Matches(modulePath) {
				continue
			}
			matches = append(matches, repoRoot.ToTilde(dt.OrFullPath))
			break
		}
	}
	err = CombineErrs(errs)

	if err != nil {
		err = WithErr(err, "scan_dir", scanDir, "repo_dir", currentRepo)
	}
	return matches, err
}
