package squirecfg

// RepoConfig represents the structure of .squire/config.json
type RepoConfig struct {
	Modules map[string]struct {
		Name string   `json:"name"`
		Kind []string `json:"kind"`
	} `json:"modules"`
	Requires []struct {
		Path string `json:"path"`
	} `json:"requires,omitempty"`
}
