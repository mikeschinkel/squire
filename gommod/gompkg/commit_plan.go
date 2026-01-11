package gompkg

import (
	"errors"
	"time"

	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/gomion/gommod/gitutils"
	"github.com/mikeschinkel/gomion/gommod/gomcfg"
	"github.com/mikeschinkel/gomion/gommod/gomion"
)

// CommitPlanMap is a map of file paths to their disposition strings
type CommitPlanMap map[dt.RelFilepath]dt.Identifier

// CommitPlan represents the runtime commit plan with domain types.
// This is converted from gomcfg.CommitPlanV1 (config layer) via ParseCommitPlan.
// The disposition values are stored as strings and should be parsed to FileDisposition
// in the gomtui layer where they're used.
type CommitPlan struct {
	Version    int
	Scope      CommitScope
	ModulePath dt.RelDirPath
	Timestamp  time.Time
	CommitPlan CommitPlanMap
}

// CommitScope indicates whether the commit plan is for a module or entire repo
type CommitScope string

const (
	ModuleScope CommitScope = "module"
	RepoScope   CommitScope = "repo"
)

// ParseCommitPlan converts gomcfg.CommitPlanV1 (scalar types) to gompkg.CommitPlan (domain types).
// Validates and converts:
// - timestamp string → time.Time
// - module_path string → dt.RelDirPath
// - commit_plan map[string]string → CommitPlanMap (string values preserved)
// - scope string → CommitScope enum
func ParseCommitPlan(cfg gomcfg.CommitPlanV1) (plan CommitPlan, err error) {
	// Parse timestamp
	plan.Timestamp, err = time.Parse(time.RFC3339, cfg.Timestamp)
	if err != nil {
		err = NewErr(ErrInvalidCommitPlan, StringKV("field", "timestamp"), err)
		goto end
	}

	// Parse scope
	switch cfg.Scope {
	case string(ModuleScope):
		plan.Scope = ModuleScope
	case string(RepoScope):
		plan.Scope = RepoScope
	default:
		err = NewErr(
			ErrInvalidCommitPlan,
			StringKV("field", "scope"),
			StringKV("value", cfg.Scope),
		)
		goto end
	}

	// Parse module path
	if cfg.ModulePath != "" {
		plan.ModulePath = dt.RelDirPath(cfg.ModulePath)
	}

	// Parse commit plan map (keep as strings - will be parsed to FileDisposition in gomtui)
	plan.CommitPlan = make(CommitPlanMap, len(cfg.CommitPlan))
	for path, disp := range cfg.CommitPlan {
		plan.CommitPlan[dt.RelFilepath(path)] = dt.Identifier(disp)
	}

	plan.Version = cfg.Version

end:
	if err != nil {
		err = WithErr(err, ErrFailedToLoadCommitPlan)
	}
	return plan, err
}

// ToConfigV1 converts runtime CommitPlan to config layer CommitPlanV1 (scalar types only).
func (cp *CommitPlan) ToConfigV1() gomcfg.CommitPlanV1 {
	cfg := gomcfg.CommitPlanV1{
		Version:    cp.Version,
		Scope:      string(cp.Scope),
		ModulePath: string(cp.ModulePath),
		Timestamp:  cp.Timestamp.Format(time.RFC3339),
		CommitPlan: make(map[string]string, len(cp.CommitPlan)),
	}

	// Convert map[dt.RelFilepath]string → map[string]string
	for path, disp := range cp.CommitPlan {
		cfg.CommitPlan[string(path)] = string(disp)
	}

	return cfg
}

// Save persists commit plan using InfoStore to .git/info/gomion/commit-plan.json
func (cp *CommitPlan) Save(repoRoot dt.DirPath) (err error) {
	var gomionDir dt.DirPath
	var exists bool
	var store *gitutils.InfoStore

	// Ensure .git/info/gomion directory exists
	gomionDir = dt.DirPathJoin(repoRoot, ".git/info/gomion")
	exists, err = gomionDir.Exists()
	if err != nil {
		err = NewErr(dt.ErrFileSystem, gomionDir.ErrKV(), err)
		goto end
	}
	if !exists {
		err = gomionDir.MkdirAll(0755)
		if err != nil {
			err = NewErr(dt.ErrFailedtoCreateDir, gomionDir.ErrKV(), err)
			goto end
		}
	}

	store = gitutils.NewInfoStore(repoRoot, gomion.CommitPlanFile)
	err = store.SaveJSON(cp.ToConfigV1())
	if err != nil {
		err = WithErr(err, ErrFailedToSaveCommitPlan, store.ErrKV())
	}

end:
	return err
}

// LoadCommitPlan loads commit plan from .git/info/gomion/commit-plan.json
// Returns nil plan and nil error if file doesn't exist (not an error condition).
func LoadCommitPlan(repoRoot dt.DirPath) (plan *CommitPlan, err error) {
	var cfg gomcfg.CommitPlanV1
	var parsedPlan CommitPlan
	var store *gitutils.InfoStore

	store = gitutils.NewInfoStore(repoRoot, gomion.CommitPlanFile)

	err = store.LoadJSON(&cfg)
	if errors.Is(err, dt.ErrFileNotExist) {
		// If file doesn't exist, that's not an error - return nil plan
		err = nil
		goto end
	}
	if err != nil {
		goto end
	}

	parsedPlan, err = ParseCommitPlan(cfg)
	if err != nil {
		err = NewErr(ErrInvalidCommitPlan, err)
		goto end
	}
	plan = &parsedPlan

end:
	if err != nil {
		err = WithErr(err, ErrFailedToLoadCommitPlan, store.ErrKV())
	}
	return plan, err
}
