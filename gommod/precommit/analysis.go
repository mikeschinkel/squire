package precommit

import (
	"context"
	"io"

	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/go-dt/dtx"
)

// AnalyzeWithCache runs pre-commit analysis with caching support
// It tries to load from cache first, and if not found, runs fresh analysis
func AnalyzeWithCache(ctx context.Context, moduleDir dt.DirPath, cacheKey string, writer io.Writer) (results *Results, err error) {
	var analysisResult Results

	// Try to load from cache first
	results, err = LoadPersistedResult(cacheKey)
	if err == nil {
		if writer != nil {
			dtx.Fprintf(writer, "Using cached analysis results\n\n")
		}
		goto end
	}

	// Run fresh analysis
	analysisResult, err = Analyze(ctx, AnalyzeArgs{
		ModuleDir: moduleDir,
		CacheKey:  cacheKey,
	})
	if err != nil {
		goto end
	}

	results = &analysisResult

end:
	return results, err
}
