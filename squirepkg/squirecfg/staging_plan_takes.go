package squirecfg

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mikeschinkel/go-cfgstore"
	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/squire/squirepkg/squire"
)

// StagingPlanTakes represents AI-generated grouping suggestions
// Stores 3 different "takes" on how to group changes
// Cached in ~/.cache/squire/analysis/{key}-takes.json
type StagingPlanTakes struct {
	CacheKey  string            `json:"cache_key"` // Hash of analysis input
	Timestamp time.Time         `json:"timestamp"` // When takes were generated
	Takes     []StagingPlanTake `json:"takes"`     // 3 different perspectives
}

// StagingPlanTake represents one AI perspective on grouping changes
type StagingPlanTake struct {
	Number int         `json:"number"` // 1, 2, or 3
	Theme  string      `json:"theme"`  // e.g., "By Feature", "By Layer", "By Risk"
	Groups []TakeGroup `json:"groups"` // Suggested groups
}

// TakeGroup represents a suggested group within a take
type TakeGroup struct {
	Name      string           `json:"name"`      // Group name
	Rationale string           `json:"rationale"` // Why these files belong together
	Files     []dt.RelFilepath `json:"files"`     // File-level grouping (user refines to lines)
}

// SaveStagingPlanTakes saves grouping takes to ~/.cache/squire/analysis/{key}-takes.json
func SaveStagingPlanTakes(cacheKey string, takes *StagingPlanTakes) (err error) {
	var cacheFile dt.Filepath
	var data []byte

	// Get cache directory (~/.cache/squire/analysis/)
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

// LoadStagingPlanTakes loads grouping takes from ~/.cache/squire/analysis/{key}-takes.json
func LoadStagingPlanTakes(cacheKey string) (takes *StagingPlanTakes, err error) {
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
	takes = &StagingPlanTakes{}
	err = json.Unmarshal(data, takes)
	if err != nil {
		err = fmt.Errorf("failed to unmarshal staging plan takes: %w", err)
		goto end
	}

end:
	return takes, err
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
	// Get cache directory (~/.cache/squire/analysis/)
	cd, err = cfgstore.GetAppCacheDir(squire.ConfigSlug, "analysis")
	if err != nil {
		goto end
	}
	cf = dt.FilepathJoin(cd, cacheKey+"-takes.json")
end:
	return cf, err
}
