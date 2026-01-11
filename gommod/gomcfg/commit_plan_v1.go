package gomcfg

// CommitPlanV1 represents a saved commit plan with scalar types only for JSON serialization.
// This is the config layer that uses only basic types (strings, ints) that marshal cleanly to JSON.
// It is converted to gompkg.CommitPlan (runtime layer) via ParseCommitPlan for use in the application.
type CommitPlanV1 struct {
	// Version is the schema version for this commit plan format
	Version int `json:"version"`

	// Scope indicates whether this plan is for a "module" or "repo"
	Scope string `json:"scope"`

	// ModulePath is the relative path to the module (if scope is "module")
	ModulePath string `json:"module_path,omitempty"`

	// Timestamp is when this commit plan was last saved (RFC3339 format)
	Timestamp string `json:"timestamp"`

	// CommitPlan maps file paths to disposition labels (lowercase)
	// Example: "gommod/gomtui/file.go" → "commit"
	CommitPlan map[string]string `json:"commit_plan"`
}
