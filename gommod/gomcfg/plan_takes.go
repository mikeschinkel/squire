package gomcfg

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"github.com/mikeschinkel/go-cfgstore"
	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/gomion/gommod/gomion"
)

// PlanTakes represents AI-generated grouping suggestions (JSON serialization)
// Stores 3 different "takes" on how to group changes
// Cached in ~/.cache/gomion/analysis/{key}-takes.json
// NOTE: Uses only scalar types for JSON serialization
type PlanTakes struct {
	CacheKey  string     `json:"cache_key"` // Hash of analysis input
	Timestamp string     `json:"timestamp"` // RFC3339 timestamp
	Takes     []PlanTake `json:"takes"`     // 3 different perspectives
}

// PlanTake represents one AI perspective on grouping changes (JSON serialization)
type PlanTake struct {
	Number int         `json:"number"` // 1, 2, or 3
	Theme  string      `json:"theme"`  // e.g., "By Feature", "By Layer", "By Risk"
	Groups []ChangeSet `json:"groups"` // Suggested groups
}

// ChangeSet represents a suggested group within a take (JSON serialization)
type ChangeSet struct {
	Name      string   `json:"name"`      // Group name
	Rationale string   `json:"rationale"` // Why these files belong together
	Files     []string `json:"files"`     // Relative file paths
}

// SavePlanTakes saves grouping takes to ~/.cache/gomion/analysis/{key}-takes.json
func SavePlanTakes(cacheKey string, takes *PlanTakes) (err error) {
	var cacheFile dt.Filepath
	var data []byte

	// Get cache directory (~/.cache/gomion/analysis/)
	cacheFile, err = getAnalysisCacheFile(cacheKey)
	if err != nil {
		goto end
	}

	// Create cache directory
	err = cacheFile.Dir().MkdirAll(0755)
	if err != nil {
		err = fmt.Errorf("failed to create cache directory: %w", err)
		goto end
	}

	// Marshal to JSON
	data, err = json.MarshalIndent(takes, "", "  ")
	if err != nil {
		err = fmt.Errorf("failed to marshal staging plan takes: %w", err)
		goto end
	}

	// Write to file
	err = cacheFile.WriteFile(data, 0644)
	if err != nil {
		err = fmt.Errorf("failed to write staging plan takes file: %w", err)
		goto end
	}

end:
	return err
}

// LoadPlanTakes loads grouping takes from ~/.cache/gomion/analysis/{key}-takes.json
func LoadPlanTakes(cacheKey string) (takes *PlanTakes, err error) {
	var cacheFile dt.Filepath
	var data []byte

	// Get cache directory
	cacheFile, err = getAnalysisCacheFile(cacheKey)
	if err != nil {
		goto end
	}

	// Read file
	data, err = cacheFile.ReadFile()
	if err != nil {
		err = fmt.Errorf("failed to read staging plan takes file: %w", err)
		goto end
	}

	// Unmarshal
	takes = &PlanTakes{}
	err = json.Unmarshal(data, takes)
	if err != nil {
		err = fmt.Errorf("failed to unmarshal staging plan takes: %w", err)
		goto end
	}

end:
	return takes, err
}

// ClearPlanTakes deletes the cached staging plan takes for the given cache key
func ClearPlanTakes(cacheKey string) (err error) {
	var cacheFile dt.Filepath

	// Get cache file path
	cacheFile, err = getAnalysisCacheFile(cacheKey)
	if err != nil {
		goto end
	}

	// Delete the file (ignore error if file doesn't exist)
	err = cacheFile.Remove()
	if err != nil {
		// Check if it's a "file not found" error - that's not really an error
		exists, existsErr := cacheFile.Exists()
		if existsErr == nil && !exists {
			err = nil // File doesn't exist, which is fine
		} else {
			err = fmt.Errorf("failed to remove staging plan takes file: %w", err)
		}
	}

end:
	return err
}

// ComputeAnalysisCacheKey computes a cache key for analysis results
// Based on the files and their content
func ComputeAnalysisCacheKey(files []dt.RelFilepath, analysisInput string) (cacheKey string) {
	h := sha256.New()

	// Hash file paths
	for _, f := range files {
		h.Write([]byte(f))
		h.Write([]byte("\n"))
	}

	// Hash analysis input
	h.Write([]byte(analysisInput))

	cacheKey = fmt.Sprintf("%x", h.Sum(nil))[:16] // Use first 16 chars
	return cacheKey
}

func getAnalysisCacheFile(cacheKey string) (cf dt.Filepath, err error) {
	var cd dt.DirPath
	// Get cache directory (~/.cache/gomion/analysis/)
	cd, err = cfgstore.GetAppCacheDir(gomion.ConfigSlug, "analysis")
	if err != nil {
		goto end
	}
	cf = dt.FilepathJoin(cd, cacheKey+"-takes.json")
end:
	return cf, err
}
