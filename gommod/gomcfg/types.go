package gomcfg

// RepoConfig represents the structure of .gomion/config.json
type RepoConfig struct {
	Modules map[string]struct {
		Name string   `json:"name"`
		Kind []string `json:"kind"`
	} `json:"modules"`
	Requires []struct {
		Path string `json:"path"`
	} `json:"requires,omitempty"`
}

// InputData mirrors the JSON shape provided to GRU via --input.
// All fields use basic scalar/string types to match the serialized format.
type InputData struct {
	ModuleDir     string        `json:"module_dir"`
	GitDiffOutput string        `json:"git_diff_output"`
	ExistingPlans []StagingPlan `json:"existing_plans"`
	AITakes       *PlanTakes    `json:"ai_takes"`
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
