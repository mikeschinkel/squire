package gompkg

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/mikeschinkel/go-cfgstore"
	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/gomion/gommod/gomion"
)

// StagingPlan represents a group of related changes to be committed together
// Replaces the old "Group" concept with clearer naming
type StagingPlan struct {
	ID          dt.Identifier    `json:"id"`          // UUID
	Name        string           `json:"name"`        // User-friendly name
	Description string           `json:"description"` // Optional description
	Created     time.Time        `json:"created"`     // Creation timestamp
	Modified    time.Time        `json:"modified"`    // Last modification timestamp
	Files       []FilePatchRange `json:"files"`       // Files with line-level ranges
	Suggested   bool             `json:"suggested"`   // True if AI-generated, false if user-created
	TakeNumber  int              `json:"take_number"` // 1-3 for AI takes, 0 for user-created
	IsDefault   bool             `json:"is_default"`  // True if this is the auto-generated default plan
}

// FilePatchRange represents a file with specific line ranges to include
// Stores hunk headers and context to handle line number shifts
type FilePatchRange struct {
	Path     dt.RelFilepath `json:"path"`      // Relative path from repo root
	Hunks    []HunkHeader   `json:"hunks"`     // Line-level ranges with context
	AllLines bool           `json:"all_lines"` // If true, include entire file (hunks ignored)
}

// HunkHeader represents a git diff hunk with context lines
// Stores the actual diff header and context to survive line number shifts
// Based on ChatGPT advice: store headers + context, not just line numbers
type HunkHeader struct {
	Header        string   `json:"header"`         // The @@ -old +new @@ line from git diff
	ContextBefore []string `json:"context_before"` // Context lines before the change
	ContextAfter  []string `json:"context_after"`  // Context lines after the change
	OldStart      int      `json:"old_start"`      // Starting line in old file
	OldCount      int      `json:"old_count"`      // Number of lines in old file
	NewStart      int      `json:"new_start"`      // Starting line in new file
	NewCount      int      `json:"new_count"`      // Number of lines in new file
}

type PlanStore struct {
	cfgstore.ConfigStore
	ModuleDir dt.DirPath
}

const StagingPlansPath = "plans"

// NewPlanStore instantiates a new plan store object
func NewPlanStore(modDir dt.DirPath, spID dt.Identifier) *PlanStore {
	return &PlanStore{
		ModuleDir: modDir,
		ConfigStore: cfgstore.NewConfigStore(cfgstore.CustomConfigDirType, cfgstore.ConfigStoreArgs{
			ConfigSlug:  gomion.ConfigSlug,
			RelFilepath: dt.RelFilepathJoin(StagingPlansPath, spID+".json"),
			DirsProvider: cfgstore.DefaultDirsProviderWithArgs(cfgstore.DirsProviderArgs{
				CustomDirPath: modDir,
			}),
		}),
	}
}

// NewStagingPlan instantiates a new staging plan object
func NewStagingPlan(name string) (sp *StagingPlan) {
	now := time.Now()
	sp = &StagingPlan{
		ID:       generatePlanID(),
		Name:     name,
		Created:  now,
		Modified: now,
		Files:    make([]FilePatchRange, 0),
	}
	return sp
}

// Save saves the staging plan to .gomion/plans/{id}.json
func (store PlanStore) Save(sp *StagingPlan) (err error) {
	// Update Modified timestamp
	sp.Modified = time.Now()

	err = store.SaveJSON(sp)
	if err != nil {
		err = fmt.Errorf("failed to write staging plan file: %w", err)
		goto end
	}

end:
	return err
}

// LoadStagingPlan loads a staging plan from .gomion/plans/{id}.json
func LoadStagingPlan(modDir dt.DirPath, id dt.Identifier) (sp *StagingPlan, store *PlanStore, err error) {
	store = NewPlanStore(modDir, id)
	sp = &StagingPlan{}
	err = store.LoadJSON(sp)
	if err != nil {
		goto end
	}
end:
	return sp, store, err
}

// ListActive lists all active staging plans
func (store PlanStore) ListActive() (plans []*StagingPlan, err error) {
	var entries []os.DirEntry
	var plan *StagingPlan

	// Get config directory
	configDir, err := store.ConfigDir()
	if err != nil {
		goto end
	}

	// Read directory
	entries, err = configDir.ReadDir()
	if err != nil {
		if os.IsNotExist(err) {
			// No plans directory yet - return empty list
			err = nil
			goto end
		}
		err = fmt.Errorf("failed to read plans directory: %w", err)
		goto end
	}

	plans = make([]*StagingPlan, 0, len(entries))

	// Load each plan
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Only process .json files
		if filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		// Extract ID from filename
		id := dt.Identifier(entry.Name()[:len(entry.Name())-5]) // Remove .json

		plan, _, err = LoadStagingPlan(store.ModuleDir, id)
		if err != nil {
			// Skip invalid plans, continue loading others
			continue
		}

		plans = append(plans, plan)
	}

end:
	return plans, err
}

// ListStagingPlans lists all staging plans in .gomion/plans/
func ListStagingPlans(moduleDir dt.DirPath) (plans []*StagingPlan, err error) {
	store := NewPlanStore(moduleDir, "")
	plans, err = store.ListActive()
	return plans, err
}

// Delete deletes a staging plan
func (store PlanStore) Delete(sp *StagingPlan) (err error) {
	var planFile dt.Filepath

	fp := store.GetRelFilepath()
	dp := fp.Dir()
	spId := sp.ID + ".json"
	planFile = dt.FilepathJoin(dp, spId)
	err = planFile.Remove()
	if cfgstore.NoFileOrDirErr(err) {
		err = nil
	}
	if err != nil {
		err = NewErr(dt.ErrFailedToRemoveFile, planFile.ErrKV(), err)
	}

	return err
}

// DeleteStagingPlan deletes a staging plan from .gomion/plans/{id}.json
func DeleteStagingPlan(moduleDir dt.DirPath, id dt.Identifier) (err error) {
	sp, store, err := LoadStagingPlan(moduleDir, id)
	if err != nil {
		goto end
	}
	err = store.Delete(sp)
end:
	return err
}

// generatePlanID generates a unique ID for a staging plan
// Uses timestamp + random component
func generatePlanID() dt.Identifier {
	return dt.Identifier(fmt.Sprintf("plan-%d", time.Now().Unix()))
}
