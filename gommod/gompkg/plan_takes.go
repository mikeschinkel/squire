package gompkg

import (
	"time"

	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/gomion/gommod/gomcfg"
)

// PlanTakes represents AI-generated grouping suggestions (general-purpose)
// Uses validated types (time.Time, dt.RelFilepath) for runtime use
// Convert from gomcfg.PlanTakes via ParsePlanTakes()
type PlanTakes struct {
	CacheKey  string
	Timestamp time.Time
	Takes     []PlanTake
}

// PlanTake represents one AI perspective on grouping changes (general-purpose)
type PlanTake struct {
	Number    int
	Theme     string
	ChangeSet []ChangeSet
}

// ChangeSet represents a suggested group within a take (general-purpose)
type ChangeSet struct {
	Name      string
	Rationale string
	Files     []dt.RelFilepath
}

// ParsePlanTakes converts gomcfg.PlanTakes to gompkg.PlanTakes
func ParsePlanTakes(cfg gomcfg.PlanTakes) (takes *PlanTakes, err error) {
	var ts time.Time

	// Parse RFC3339 timestamp
	ts, err = time.Parse(time.RFC3339, cfg.Timestamp)
	if err != nil {
		goto end
	}

	takes = &PlanTakes{
		CacheKey:  cfg.CacheKey,
		Timestamp: ts,
		Takes:     make([]PlanTake, len(cfg.Takes)),
	}

	// Parse each take
	for i, cfgTake := range cfg.Takes {
		takes.Takes[i], err = ParsePlanTake(cfgTake)
		if err != nil {
			goto end
		}
	}

end:
	return takes, err
}

// ParsePlanTake converts gomcfg.PlanTake to gompkg.PlanTake
func ParsePlanTake(cfg gomcfg.PlanTake) (take PlanTake, err error) {
	take = PlanTake{
		Number:    cfg.Number,
		Theme:     cfg.Theme,
		ChangeSet: make([]ChangeSet, len(cfg.Groups)),
	}

	// Parse each change set
	for i, cfgGroup := range cfg.Groups {
		take.ChangeSet[i], err = ParseChangeSet(cfgGroup)
		if err != nil {
			goto end
		}
	}

end:
	return take, err
}

// ParseChangeSet converts gomcfg.ChangeSet to gompkg.ChangeSet
func ParseChangeSet(cfg gomcfg.ChangeSet) (cs ChangeSet, err error) {
	var files []dt.RelFilepath

	files, err = ParseRelFilepaths(cfg.Files)
	if err != nil {
		goto end
	}

	cs.Name = cfg.Name
	cs.Rationale = cfg.Rationale
	cs.Files = files

end:
	return cs, err
}

// ParseRelFilepaths converts []string to []dt.RelFilepath
func ParseRelFilepaths(paths []string) (files []dt.RelFilepath, err error) {
	var fp dt.RelFilepath

	files = make([]dt.RelFilepath, 0, len(paths))
	for i, p := range paths {
		fp, err = dt.ParseRelFilepath(p)
		if err != nil {
			err = NewErr("parse_filepath", err, "index", i, "path", p)
			goto end
		}
		files = append(files, fp)
	}

end:
	return files, err
}

// ToCfg converts gompkg.PlanTakes to gomcfg.PlanTakes for JSON serialization
func (pt *PlanTakes) ToCfg() gomcfg.PlanTakes {
	cfg := gomcfg.PlanTakes{
		CacheKey:  pt.CacheKey,
		Timestamp: pt.Timestamp.Format(time.RFC3339),
		Takes:     make([]gomcfg.PlanTake, len(pt.Takes)),
	}

	for i, take := range pt.Takes {
		cfg.Takes[i] = take.ToCfg()
	}

	return cfg
}

// ToCfg converts gompkg.PlanTake to gomcfg.PlanTake
func (pt *PlanTake) ToCfg() gomcfg.PlanTake {
	cfg := gomcfg.PlanTake{
		Number: pt.Number,
		Theme:  pt.Theme,
		Groups: make([]gomcfg.ChangeSet, len(pt.ChangeSet)),
	}

	for i, group := range pt.ChangeSet {
		cfg.Groups[i] = group.ToCfg()
	}

	return cfg
}

// ToCfg converts gompkg.ChangeSet to gomcfg.ChangeSet
func (cs *ChangeSet) ToCfg() gomcfg.ChangeSet {
	cfg := gomcfg.ChangeSet{
		Name:      cs.Name,
		Rationale: cs.Rationale,
		Files:     make([]string, len(cs.Files)),
	}

	for i, file := range cs.Files {
		cfg.Files[i] = string(file)
	}

	return cfg
}

// SavePlanTakes saves PlanTakes to cache (converts to gomcfg and delegates)
func SavePlanTakes(cacheKey string, takes *PlanTakes) (err error) {
	cfg := takes.ToCfg()
	err = gomcfg.SavePlanTakes(cacheKey, &cfg)
	return err
}

// LoadPlanTakes loads PlanTakes from cache (loads gomcfg and converts)
func LoadPlanTakes(cacheKey string) (takes *PlanTakes, err error) {
	var cfgTakes *gomcfg.PlanTakes

	cfgTakes, err = gomcfg.LoadPlanTakes(cacheKey)
	if err != nil {
		goto end
	}

	takes, err = ParsePlanTakes(*cfgTakes)

end:
	return takes, err
}

// ClearPlanTakes deletes cached PlanTakes (delegates to gomcfg)
func ClearPlanTakes(cacheKey string) (err error) {
	err = gomcfg.ClearPlanTakes(cacheKey)
	return err
}

// ComputeAnalysisCacheKey computes a cache key (delegates to gomcfg)
func ComputeAnalysisCacheKey(files []dt.RelFilepath, analysisInput string) (cacheKey string) {
	cacheKey = gomcfg.ComputeAnalysisCacheKey(files, analysisInput)
	return cacheKey
}
