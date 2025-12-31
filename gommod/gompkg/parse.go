package gompkg

import (
	"fmt"
	"time"

	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/gomion/gommod/gomcfg"
)

// ParseInputData converts config-layer input into domain types, validating along the way.
func ParseInputData(cfg gomcfg.InputData) (data InputData, err error) {
	var moduleDir dt.DirPath
	var plans []StagingPlan
	var takes *PlanTakes

	plans, err = ParseStagingPlans(cfg.ExistingPlans)
	if err != nil {
		goto end
	}

	if cfg.AITakes != nil {
		takes, err = ParsePlanTakes(*cfg.AITakes)
		if err != nil {
			goto end
		}
	}

	data.ModuleDir = moduleDir
	data.GitDiffOutput = cfg.GitDiffOutput
	data.ExistingPlans = plans
	data.AITakes = takes

end:
	return data, err
}

// ParseOutputData converts config-layer output into domain types.
func ParseOutputData(cfg gomcfg.OutputData) (data OutputData, err error) {
	var plans []StagingPlan

	plans, err = ParseStagingPlans(cfg.Plans)
	if err != nil {
		goto end
	}

	data.Plans = plans

end:
	return data, err
}

func ParseStagingPlans(cfgPlans []gomcfg.StagingPlan) (plans []StagingPlan, err error) {
	var plan StagingPlan

	plans = make([]StagingPlan, 0, len(cfgPlans))
	for i, cfgPlan := range cfgPlans {
		plan, err = ParseStagingPlan(cfgPlan)
		if err != nil {
			err = fmt.Errorf("existing_plans[%d]: %w", i, err)
			goto end
		}
		plans = append(plans, plan)
	}

end:
	return plans, err
}

func ParseStagingPlan(cfgPlan gomcfg.StagingPlan) (plan StagingPlan, err error) {
	var id dt.Identifier
	var created time.Time
	var modified time.Time
	var files []FilePatchRange

	id, err = dt.ParseIdentifier(cfgPlan.ID)
	if err != nil {
		err = fmt.Errorf("id: %w", err)
		goto end
	}

	created, err = ParseRFC3339(cfgPlan.Created)
	if err != nil {
		err = fmt.Errorf("created: %w", err)
		goto end
	}

	modified, err = ParseRFC3339(cfgPlan.Modified)
	if err != nil {
		err = fmt.Errorf("modified: %w", err)
		goto end
	}

	files, err = ParseFilePatchRanges(cfgPlan.Files)
	if err != nil {
		goto end
	}

	plan.ID = id
	plan.Name = cfgPlan.Name
	plan.Description = cfgPlan.Description
	plan.Created = created
	plan.Modified = modified
	plan.Files = files
	plan.Suggested = cfgPlan.Suggested
	plan.TakeNumber = cfgPlan.TakeNumber
	plan.IsDefault = cfgPlan.IsDefault

end:
	return plan, err
}

func ParseFilePatchRanges(cfgRanges []gomcfg.FilePatchRange) (ranges []FilePatchRange, err error) {
	var rng FilePatchRange

	ranges = make([]FilePatchRange, 0, len(cfgRanges))
	for i, cfgRange := range cfgRanges {
		rng, err = ParseFilePatchRange(cfgRange)
		if err != nil {
			err = fmt.Errorf("files[%d]: %w", i, err)
			goto end
		}
		ranges = append(ranges, rng)
	}

end:
	return ranges, err
}

func ParseFilePatchRange(cfgRange gomcfg.FilePatchRange) (rng FilePatchRange, err error) {
	var path dt.RelFilepath
	var hunks []HunkHeader

	path, err = dt.ParseRelFilepath(cfgRange.Path)
	if err != nil {
		err = fmt.Errorf("path: %w", err)
		goto end
	}

	hunks, err = ParseHunks(cfgRange.Hunks)
	if err != nil {
		goto end
	}

	rng.Path = path
	rng.Hunks = hunks
	rng.AllLines = cfgRange.AllLines

end:
	return rng, err
}

func ParseHunks(cfgHunks []gomcfg.HunkHeader) (hunks []HunkHeader, err error) {
	var hunk HunkHeader

	hunks = make([]HunkHeader, 0, len(cfgHunks))
	for i, cfgHunk := range cfgHunks {
		hunk, err = ParseHunk(cfgHunk)
		if err != nil {
			err = fmt.Errorf("hunks[%d]: %w", i, err)
			goto end
		}
		hunks = append(hunks, hunk)
	}

end:
	return hunks, err
}

func ParseHunk(cfgHunk gomcfg.HunkHeader) (hunk HunkHeader, err error) {
	hunk.Header = cfgHunk.Header
	hunk.ContextBefore = append(hunk.ContextBefore, cfgHunk.ContextBefore...)
	hunk.ContextAfter = append(hunk.ContextAfter, cfgHunk.ContextAfter...)
	hunk.OldStart = cfgHunk.OldStart
	hunk.OldCount = cfgHunk.OldCount
	hunk.NewStart = cfgHunk.NewStart
	hunk.NewCount = cfgHunk.NewCount
	return hunk, err
}

func ParseRFC3339(value string) (t time.Time, err error) {
	if value == "" {
		err = fmt.Errorf("ts is empty")
		goto end
	}
	t, err = time.Parse(time.RFC3339, value)
end:
	return t, err
}
