package gommod

import (
	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/gomion/gommod/gomcfg"
	"github.com/mikeschinkel/gomion/gommod/gompkg"
)

func ParseConfig(cfg *gomcfg.RootConfigV1, args gompkg.ConfigArgs) (c *gompkg.Config, err error) {
	var scanDirs []dt.DirPath
	var modSpecs []gompkg.ModuleSpec

	scanDirs, err = dt.ParseDirPaths(cfg.ScanDirs)
	if err != nil {
		goto end
	}

	modSpecs, err = gompkg.ParseModuleSpecs(cfg.ModuleSpecs)
	if err != nil {
		goto end
	}

	c = &gompkg.Config{
		Options:     args.Options,
		ScanDirs:    scanDirs,
		ModuleSpecs: modSpecs,
		Logger:      args.Logger,
		Writer:      args.Writer,
	}
end:
	return c, err
}
