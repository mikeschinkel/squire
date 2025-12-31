package gompkg

import (
	"github.com/mikeschinkel/go-cfgstore"
	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/gomion/gommod/gomion"
)

// ProjectConfigStore provides a project config store for gomion at the specified project directory.
// This is used to load project configs from repos in other directories.
// TODO Verify that this can be unit-tested given its hardcoded DirsProvider

func ProjectConfigStore(projectDir dt.DirPath) (cs cfgstore.ConfigStore) {
	return cfgstore.NewConfigStore(cfgstore.ProjectConfigDirType, cfgstore.ConfigStoreArgs{
		ConfigSlug:  gomion.ConfigSlug,
		RelFilepath: gomion.ConfigFile,
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
	fp, err = cfgstore.ProjectConfigFilepath(gomion.ConfigSlug, gomion.ConfigFile)
	if err != nil {
		goto end
	}
	projectFilepaths[projectDir] = fp
end:
	return fp, err
}
