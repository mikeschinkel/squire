package gru

import (
	"github.com/mikeschinkel/go-dt"
)

var (
	// versionString is set at build time by GoReleaser via -X flag.
	// Must be a plain string type for -X to work.
	versionString = "v0.0.0-dev"

	// Version is the application version exposed to the rest of the application.
	Version = dt.Version(versionString)
)

const (
	// AppName is the human-readable name of the application.
	AppName                 = "Gru"
	AppDescr                = "Gru staging utility for Go modules"
	AppSlug  dt.PathSegment = "gru"

	// ConfigSlug provides the directory under ~/.config/ where configuration will be
	// stored. This is not squire-cli as everything Squire goes under the one location.
	ConfigSlug dt.PathSegment = "gru"

	// ConfigFile is the path for where the config file will be stored in the config
	// directory, e.g. ~/.config/Squire/cli.json
	ConfigFile dt.RelFilepath = "config.json"

	// ExeName is just Squire not Squirecli or Squire-cli as those are redundant, and
	// Squire should be the only CLI executable we put on a user's machine; everything
	// else gets loaded or run by this one executable. Not that the other packages
	// have their own ExeName values, but that is merely for our own convenince and
	// we do not expect to distribute those executables.
	ExeName dt.Filename = "gru"

	LogPath dt.PathSegments = "logs"

	// GitHubRepoURL provides the GitHub repo for this project for use in error messages
	GitHubRepoURL dt.URL = "https://github.com/mikeschinkel/squire"

	DefaultInputFile  dt.Filename = "input.json"
	DefaultOutputFile dt.Filename = "output.json"
)

const (
	InfoURL           = GitHubRepoURL
	ProjectConfigPath = "." + ConfigSlug
)

var (
	LogFile   = dt.RelFilepathJoin(LogPath, string(AppSlug)+".log")
	ExtraInfo = map[string]any{
		"github_repo_url": GitHubRepoURL,
	}
)
