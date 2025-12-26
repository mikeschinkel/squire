package squiresvc

import (
	"github.com/mikeschinkel/go-cfgstore"
	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/squire/squirepkg/squire"
)

// ProjectConfigStore provides a project config store for squire at the specified project directory.
// This is used to load project configs from repos in other directories.
// TODO Verify that this can be unit-tested given its hardcoded DirsProvider

func ProjectConfigStore(projectDir dt.DirPath) (cs cfgstore.ConfigStore) {
	return cfgstore.NewConfigStore(cfgstore.ProjectConfigDirType, cfgstore.ConfigStoreArgs{
		ConfigSlug:  squire.ConfigSlug,
		RelFilepath: squire.ConfigFile,
		DirsProvider: &cfgstore.DirsProvider{
			ProjectDirFunc: func() (dt.DirPath, error) {
				return projectDir, nil
			},
		},
	})
}

var projectFilepaths = make(map[dt.DirPath]dt.Filepath)

func ProjectConfigFilepath(projectDir dt.DirPath) (fp dt.Filepath, err error) {
	var ok bool
	fp, ok = projectFilepaths[projectDir]
	if ok {
		goto end
	}
	fp, err = cfgstore.ProjectConfigFilepath(squire.ConfigSlug, squire.ConfigFile)
	if err != nil {
		goto end
	}
	projectFilepaths[projectDir] = fp
end:
	return fp, err
}
