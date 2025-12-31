package gompkg

import (
	"fmt"
	"log/slog"

	"github.com/mikeschinkel/go-cfgstore"
	"github.com/mikeschinkel/go-cliutil"
	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/gomion/gommod/gomion"
)

// UpdateRepoRequiresArgs contains arguments for updating repo requires
type UpdateRepoRequiresArgs struct {
	DirArg string
	Config *Config
	Writer cliutil.Writer
	Logger *slog.Logger
}

// UpdateRepoRequiresResult contains the result of updating requires
type UpdateRepoRequiresResult struct {
	RepoRoot     dt.DirPath
	RequireCount int
	Config       *Config
}

// ModuleConfig represents a single module in the repo config
type ModuleConfig struct {
	Name  string       `json:"name"`
	Kinds []ModuleKind `json:"kinds"`
}

// RepoConfig contains the modules and their required modules
type RepoConfig struct {
	Modules  map[dt.DirPath]ModuleConfig `json:"modules"`
	Requires []RepoRequirement           `json:"requires,omitempty"`
}

// UpdateRepoRequires updates the requires field in .gomion/config.json for a repo
func UpdateRepoRequires(args *UpdateRepoRequiresArgs) (result *UpdateRepoRequiresResult, err error) {
	var dirPath dt.DirPath
	var arg string
	var repoRoot dt.DirPath
	var store cfgstore.ConfigStore
	var repoConfig *RepoConfig
	var requires []RepoRequirement
	var exists bool

	result = &UpdateRepoRequiresResult{}

	if len(args.Config.ModuleSpecs) == 0 {
		args.Writer.Printf("Please add at least one module spec pattern using: gomion project modspec add <pattern>\n")
		err = fmt.Errorf("no module_specs configured in ~/.config/gomion/config.json")
		goto end
	}

	// Get directory argument or default to current directory
	if args.DirArg == "" {
		arg = "."
	} else {
		arg = args.DirArg
	}

	// Parse and expand the directory path
	dirPath, err = dt.ParseDirPath(arg)
	if err != nil {
		err = fmt.Errorf("parsing directory path %q: %w", arg, err)
		goto end
	}

	// Find repo root from the specified directory
	repoRoot, err = FindRepoRoot(dirPath)
	if err != nil {
		err = fmt.Errorf("finding repo root from %s: %w", dirPath, err)
		goto end
	}

	// Create a config store for this repo
	store = cfgstore.NewProjectConfigStore(gomion.ConfigSlug, gomion.ConfigFile)
	store.SetConfigDir(repoRoot)

	// Check if config exists
	exists = store.Exists()
	if !exists {
		args.Writer.Printf("Repository not managed by Gomion. Run 'gomion init' first.\n")
		err = fmt.Errorf("config not found at %s", repoRoot)
		goto end
	}

	// Load existing config
	repoConfig = &RepoConfig{}
	err = store.LoadJSON(repoConfig)
	if err != nil {
		err = fmt.Errorf("loading config from %s: %w", repoRoot, err)
		goto end
	}

	// Discover requires
	requires, err = DiscoverRequires(&DiscoverRequiresArgs{
		RepoRoot: repoRoot,
		Logger:   args.Logger,
	})
	if err != nil {
		// Log warning but don't fail - requires field is optional
		args.Logger.Warn("could not discover requires", "error", err)
		err = nil
	}

	// Update the requires field
	repoConfig.Requires = requires

	// Save updated config
	err = store.SaveJSON(repoConfig)
	if err != nil {
		err = fmt.Errorf("saving config to %s: %w", repoRoot, err)
		goto end
	}

	result.RepoRoot = repoRoot
	result.RequireCount = len(requires)

end:
	return result, err
}
