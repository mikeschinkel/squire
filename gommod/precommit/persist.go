package precommit

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/go-dt/dtx"
)

// PersistResult saves analysis results to cache for later retrieval
func PersistResult(result *Results, cacheKey string) (err error) {
	var cacheDir dt.DirPath
	var cacheFile dt.Filepath
	var data []byte
	var file *os.File

	// Determine cache directory
	cacheDir, err = getCacheDir()
	if err != nil {
		err = NewErr(ErrPrecommit, "operation", "get_cache_dir", err)
		goto end
	}

	// Ensure cache directory exists
	err = os.MkdirAll(string(cacheDir), 0755)
	if err != nil {
		err = NewErr(ErrPrecommit, "operation", "mkdir_cache", err)
		goto end
	}

	// Build cache file path
	cacheFile = dt.Filepath(cacheDir.Join(cacheKey + ".json"))

	// Serialize to JSON
	data, err = json.MarshalIndent(result, "", "  ")
	if err != nil {
		err = NewErr(ErrPrecommit, "operation", "marshal_json", err)
		goto end
	}

	// Write to file
	file, err = os.Create(string(cacheFile))
	if err != nil {
		err = NewErr(ErrPrecommit, "operation", "create_cache_file", err)
		goto end
	}
	defer file.Close()

	_, err = file.Write(data)
	if err != nil {
		err = NewErr(ErrPrecommit, "operation", "write_cache_file", err)
		goto end
	}

end:
	return err
}

// LoadPersistedResult loads analysis results from cache
func LoadPersistedResult(cacheKey string) (result *Results, err error) {
	var cacheDir dt.DirPath
	var cacheFile dt.Filepath
	var data []byte

	// Determine cache directory
	cacheDir, err = getCacheDir()
	if err != nil {
		err = NewErr(ErrPrecommit, "operation", "get_cache_dir", err)
		goto end
	}

	// Build cache file path
	cacheFile = dt.Filepath(cacheDir.Join(cacheKey + ".json"))

	// Check if file exists
	_, err = os.Stat(string(cacheFile))
	if os.IsNotExist(err) {
		err = NewErr(ErrPrecommit, ErrCacheNotFound,
			"cache_key", cacheKey)
		goto end
	}
	if err != nil {
		err = NewErr(ErrPrecommit, "operation", "stat_cache_file", err)
		goto end
	}

	// Read file
	data, err = os.ReadFile(string(cacheFile))
	if err != nil {
		err = NewErr(ErrPrecommit, "operation", "read_cache_file", err)
		goto end
	}

	// Deserialize from JSON
	result = &Results{}
	err = json.Unmarshal(data, result)
	if err != nil {
		err = NewErr(ErrPrecommit, "operation", "unmarshal_json", err)
		goto end
	}

end:
	return result, err
}

// ClearPersistedResult removes cached analysis results
func ClearPersistedResult(cacheKey string) (err error) {
	var cacheDir dt.DirPath
	var cacheFile dt.Filepath

	// Determine cache directory
	cacheDir, err = getCacheDir()
	if err != nil {
		err = NewErr(ErrPrecommit, "operation", "get_cache_dir", err)
		goto end
	}

	// Build cache file path
	cacheFile = dt.Filepath(cacheDir.Join(cacheKey + ".json"))

	// Remove file (ignore if doesn't exist)
	err = os.Remove(string(cacheFile))
	if os.IsNotExist(err) {
		err = nil
	}
	if err != nil {
		err = NewErr(ErrPrecommit, "operation", "remove_cache_file", err)
		goto end
	}

end:
	return err
}

// getCacheDir returns the cache directory path
func getCacheDir() (cacheDir dt.DirPath, err error) {
	var userCacheDir string

	// Get user cache directory
	userCacheDir, err = os.UserCacheDir()
	if err != nil {
		err = NewErr(ErrPrecommit, "operation", "get_user_cache_dir", err)
		goto end
	}

	// Build gomion analysis cache path
	cacheDir = dt.DirPath(filepath.Join(userCacheDir, "gomion", "analysis"))

end:
	return cacheDir, err
}

// ComputeCacheKey generates a cache key from module directory and staged files
func ComputeCacheKey(moduleDir dt.DirPath, stagedFiles []dt.RelFilepath) string {
	hasher := sha256.New()

	// Hash module directory
	dtx.Fprintf(hasher, "%s\n", moduleDir)

	// Hash sorted staged file paths
	for _, file := range stagedFiles {
		dtx.Fprintf(hasher, "%s\n", file)
	}

	return hex.EncodeToString(hasher.Sum(nil))
}
