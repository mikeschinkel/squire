package gompkg

import (
	"errors"
	"time"

	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/go-dt/dtx"
	"github.com/mikeschinkel/gomion/gommod/gitutils"
	"github.com/mikeschinkel/gomion/gommod/gomcfg"
	"github.com/mikeschinkel/gomion/gommod/gomion"
)

// FileDispositionsMap is a map of file paths to their disposition strings
type FileDispositionsMap = *dtx.SafeMap[dt.RelFilepath, FileDisposition]

func NewFileDispositionsMap(cap int) FileDispositionsMap {
	return dtx.NewSafeMap[dt.RelFilepath, FileDisposition](cap)
}

// CommitPlan represents the runtime commit plan with domain types.
// This is converted from gomcfg.CommitPlanV1 (config layer) via ParseCommitPlan.
// The disposition values are stored as strings and should be parsed to FileDisposition
// in the gomtui layer where they're used.
type CommitPlan struct {
	ModulePath dt.RelDirPath
	FilesMap   FileDispositionsMap
	Scope      CommitScope
	Timestamp  time.Time
}

func (cp *CommitPlan) GetFileDisposition(fp dt.RelFilepath) FileDisposition {
	fd, _ := cp.FilesMap.Get(fp)
	return fd
}

func (cp *CommitPlan) SetFileDisposition(fp dt.RelFilepath, fd FileDisposition) {
	cp.FilesMap.Set(fp, fd)
}

func NewCommitPlan(modulePath dt.RelDirPath) *CommitPlan {
	scope := RepoScope
	if modulePath != "" {
		scope = ModuleScope
	}
	return &CommitPlan{
		Scope:      scope,
		ModulePath: modulePath,
		FilesMap:   NewFileDispositionsMap(0),
		Timestamp:  time.Now(),
	}
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
	var errs []error

	// Parse timestamp
	plan.Timestamp, err = time.Parse(time.RFC3339, cfg.Timestamp)
	if err != nil {
		err = NewErr(ErrInvalidCommitPlan, StringKV("field", "timestamp"), err)
		goto end
	}

	// Parse scope
	plan.Scope = RepoScope

	// Parse module path
	if cfg.ModulePath != "" {
		plan.Scope = ModuleScope
		plan.ModulePath = dt.RelDirPath(cfg.ModulePath)
	}

	// Parse commit plan map (keep as strings - will be parsed to FileDisposition in gomtui)
	plan.FilesMap = NewFileDispositionsMap(len(cfg.FilesMap))
	for path, disp := range cfg.FilesMap {
		fd, err := ParseFileDisposition(disp)
		errs = AppendErr(errs, err)
		plan.FilesMap.Set(dt.RelFilepath(path), fd)
	}
	err = CombineErrs(errs)

end:
	if err != nil {
		err = WithErr(err, ErrFailedToLoadCommitPlan)
	}
	return plan, err
}

// ToConfigV1 converts runtime CommitPlan to config layer CommitPlanV1 (scalar types only).
func (cp *CommitPlan) ToConfigV1() gomcfg.CommitPlanV1 {
	cfg := gomcfg.CommitPlanV1{
		Version:    1,
		ModulePath: string(cp.ModulePath),
		Timestamp:  cp.Timestamp.Format(time.RFC3339),
		FilesMap:   make(map[string]string, cp.FilesMap.Len()),
	}

	// Convert map[dt.RelFilepath]string → map[string]string
	for path, disp := range cp.FilesMap.Iter() {
		cfg.FilesMap[string(path)] = disp.Slug()
	}

	return cfg
}

// Save persists commit plan using InfoStore to .git/info/gomion/commit-plan.json
func (cp *CommitPlan) Save(repoRoot dt.DirPath) (err error) {
	var exists bool
	var fp dt.Filepath
	var store *gitutils.InfoStore

	// Ensure .git/info/gomion directory exists
	//
	store = gitutils.NewInfoStore(repoRoot, gomion.CommitPlanFile)
	fp, err = store.GetFilepath()
	if err != nil {
		err = NewErr(gitutils.ErrFailedToGetGitInfoFilepath, repoRoot.ErrKV(), err)
		goto end
	}
	exists, err = fp.Dir().Exists()
	if err != nil {
		err = NewErr(dt.ErrDirNotExist, fp.Dir().ErrKV(), err)
		goto end
	}
	if !exists {
		err = fp.Dir().MkdirAll(0755)
		if err != nil {
			err = NewErr(dt.ErrFailedtoCreateDir, fp.Dir().ErrKV(), err)
			goto end
		}
	}

	err = store.SaveJSON(cp.ToConfigV1())

end:
	if err != nil {
		err = WithErr(err, ErrFailedToSaveCommitPlan, store.ErrKV())
	}
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
