package gompkg

import (
	"github.com/mikeschinkel/go-dt"
)

// isRepoManaged checks if a repo has .gomion/config.json
func isRepoManaged(repoRoot dt.DirPath) (managed bool, err error) {
	var fp dt.Filepath
	fp, err = ProjectConfigFilepath(repoRoot)
	if err != nil {
		goto end
	}
	managed, err = fp.Exists()
end:
	return managed, err
}
