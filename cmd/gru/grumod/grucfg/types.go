package grucfg

// InputData mirrors the JSON shape provided to GRU via --input.
// All fields use basic scalar/string types to match the serialized format.
type InputData struct {
	ModuleDir     string            `json:"module_dir"`
	GitDiffOutput string            `json:"git_diff_output"`
	ExistingPlans []StagingPlan     `json:"existing_plans"`
	AITakes       *StagingPlanTakes `json:"ai_takes"`
}

// OutputData mirrors the JSON shape written by GRU via --output.
type OutputData struct {
	Plans []StagingPlan `json:"plans"`
}

// StagingPlan matches the config-layer representation (string/int fields).
type StagingPlan struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Created     string           `json:"created"`
	Modified    string           `json:"modified"`
	Files       []FilePatchRange `json:"files"`
	Suggested   bool             `json:"suggested"`
	TakeNumber  int              `json:"take_number"`
	IsDefault   bool             `json:"is_default"`
}

// FilePatchRange holds per-file hunk data.
type FilePatchRange struct {
	Path     string       `json:"path"`
	Hunks    []HunkHeader `json:"hunks"`
	AllLines bool         `json:"all_lines"`
}

// HunkHeader captures the parsed git hunk header plus context lines.
type HunkHeader struct {
	Header        string   `json:"header"`
	ContextBefore []string `json:"context_before"`
	ContextAfter  []string `json:"context_after"`
	OldStart      int      `json:"old_start"`
	OldCount      int      `json:"old_count"`
	NewStart      int      `json:"new_start"`
	NewCount      int      `json:"new_count"`
}

// StagingPlanTakes are AI suggestions for staging plans.
type StagingPlanTakes struct {
	CacheKey  string            `json:"cache_key"`
	Timestamp string            `json:"timestamp"`
	Takes     []StagingPlanTake `json:"takes"`
}

// StagingPlanTake represents one AI grouping perspective.
type StagingPlanTake struct {
	Number int         `json:"number"`
	Theme  string      `json:"theme"`
	Groups []TakeGroup `json:"groups"`
}

// TakeGroup groups files within a take.
type TakeGroup struct {
	Name      string   `json:"name"`
	Rationale string   `json:"rationale"`
	Files     []string `json:"files"`
}
