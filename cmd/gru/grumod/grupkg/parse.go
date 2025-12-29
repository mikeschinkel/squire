package grupkg

import (
	"fmt"
	"time"

	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/squire/gru/grumod/grucfg"
	"github.com/mikeschinkel/squire/squirepkg/squirecfg"
	"github.com/mikeschinkel/squire/squirepkg/squiresvc"
)

// ParseInputData converts config-layer input into domain types, validating along the way.
func ParseInputData(cfg grucfg.InputData) (data InputData, err error) {
	var moduleDir dt.DirPath
	var plans []squiresvc.StagingPlan
	var takes *squirecfg.StagingPlanTakes

	plans, err = ParseStagingPlans(cfg.ExistingPlans)
	if err != nil {
		goto end
	}

	if cfg.AITakes != nil {
		takes, err = ParseStagingPlanTakes(*cfg.AITakes)
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
func ParseOutputData(cfg grucfg.OutputData) (data OutputData, err error) {
	var plans []squiresvc.StagingPlan

	plans, err = ParseStagingPlans(cfg.Plans)
	if err != nil {
		goto end
	}

	data.Plans = plans

end:
	return data, err
}

func ParseStagingPlans(cfgPlans []grucfg.StagingPlan) (plans []squiresvc.StagingPlan, err error) {
	var plan squiresvc.StagingPlan

	plans = make([]squiresvc.StagingPlan, 0, len(cfgPlans))
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

func ParseStagingPlan(cfgPlan grucfg.StagingPlan) (plan squiresvc.StagingPlan, err error) {
	var id dt.Identifier
	var created time.Time
	var modified time.Time
	var files []squiresvc.FilePatchRange

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

func ParseFilePatchRanges(cfgRanges []grucfg.FilePatchRange) (ranges []squiresvc.FilePatchRange, err error) {
	var rng squiresvc.FilePatchRange

	ranges = make([]squiresvc.FilePatchRange, 0, len(cfgRanges))
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

func ParseFilePatchRange(cfgRange grucfg.FilePatchRange) (rng squiresvc.FilePatchRange, err error) {
	var path dt.RelFilepath
	var hunks []squiresvc.HunkHeader

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

func ParseHunks(cfgHunks []grucfg.HunkHeader) (hunks []squiresvc.HunkHeader, err error) {
	var hunk squiresvc.HunkHeader

	hunks = make([]squiresvc.HunkHeader, 0, len(cfgHunks))
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

func ParseHunk(cfgHunk grucfg.HunkHeader) (hunk squiresvc.HunkHeader, err error) {
	hunk.Header = cfgHunk.Header
	hunk.ContextBefore = append(hunk.ContextBefore, cfgHunk.ContextBefore...)
	hunk.ContextAfter = append(hunk.ContextAfter, cfgHunk.ContextAfter...)
	hunk.OldStart = cfgHunk.OldStart
	hunk.OldCount = cfgHunk.OldCount
	hunk.NewStart = cfgHunk.NewStart
	hunk.NewCount = cfgHunk.NewCount
	return hunk, err
}

func ParseStagingPlanTakes(cfgTakes grucfg.StagingPlanTakes) (takes *squirecfg.StagingPlanTakes, err error) {
	var ts time.Time
	var parsedTakes []squirecfg.StagingPlanTake

	ts, err = ParseRFC3339(cfgTakes.Timestamp)
	if err != nil {
		err = fmt.Errorf("timestamp: %w", err)
		goto end
	}

	parsedTakes, err = ParseAITakes(cfgTakes.Takes)
	if err != nil {
		goto end
	}

	takes = &squirecfg.StagingPlanTakes{
		CacheKey:  cfgTakes.CacheKey,
		Timestamp: ts,
		Takes:     parsedTakes,
	}

end:
	return takes, err
}

func ParseAITakes(cfgTakes []grucfg.StagingPlanTake) (takes []squirecfg.StagingPlanTake, err error) {
	var take squirecfg.StagingPlanTake

	takes = make([]squirecfg.StagingPlanTake, 0, len(cfgTakes))
	for i, cfgTake := range cfgTakes {
		take, err = ParseAITake(cfgTake)
		if err != nil {
			err = fmt.Errorf("takes[%d]: %w", i, err)
			goto end
		}
		takes = append(takes, take)
	}

end:
	return takes, err
}

func ParseAITake(cfgTake grucfg.StagingPlanTake) (take squirecfg.StagingPlanTake, err error) {
	var groups []squirecfg.TakeGroup

	groups, err = ParseTakeGroups(cfgTake.Groups)
	if err != nil {
		goto end
	}

	take.Number = cfgTake.Number
	take.Theme = cfgTake.Theme
	take.Groups = groups

end:
	return take, err
}

func ParseTakeGroups(cfgGroups []grucfg.TakeGroup) (groups []squirecfg.TakeGroup, err error) {
	var group squirecfg.TakeGroup

	groups = make([]squirecfg.TakeGroup, 0, len(cfgGroups))
	for i, cfgGroup := range cfgGroups {
		group, err = ParseTakeGroup(cfgGroup)
		if err != nil {
			err = fmt.Errorf("groups[%d]: %w", i, err)
			goto end
		}
		groups = append(groups, group)
	}

end:
	return groups, err
}

func ParseTakeGroup(cfgGroup grucfg.TakeGroup) (group squirecfg.TakeGroup, err error) {
	var files []dt.RelFilepath

	files, err = ParseRelFilepaths(cfgGroup.Files)
	if err != nil {
		err = fmt.Errorf("files: %w", err)
		goto end
	}

	group.Name = cfgGroup.Name
	group.Rationale = cfgGroup.Rationale
	group.Files = files

end:
	return group, err
}

func ParseRelFilepaths(paths []string) (files []dt.RelFilepath, err error) {
	var fp dt.RelFilepath

	files = make([]dt.RelFilepath, 0, len(paths))
	for i, p := range paths {
		fp, err = dt.ParseRelFilepath(p)
		if err != nil {
			err = fmt.Errorf("paths[%d]: %w", i, err)
			goto end
		}
		files = append(files, fp)
	}

end:
	return files, err
}

func ParseRFC3339(value string) (t time.Time, err error) {
	if value == "" {
		err = fmt.Errorf("timestamp is empty")
		goto end
	}
	t, err = time.Parse(time.RFC3339, value)
end:
	return t, err
}
