package squirepkg

import (
	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/squire/squirepkg/squirecfg"
	"github.com/mikeschinkel/squire/squirepkg/squiresvc"
)

func ParseConfig(cfg *squirecfg.RootConfigV1, args squiresvc.ConfigArgs) (c *squiresvc.Config, err error) {
	var scanDirs []dt.DirPath
	var modSpecs []squiresvc.ModuleSpec

	scanDirs, err = dt.ParseDirPaths(cfg.ScanDirs)
	if err != nil {
		goto end
	}

	modSpecs, err = squiresvc.ParseModuleSpecs(cfg.ModuleSpecs)
	if err != nil {
		goto end
	}

	c = &squiresvc.Config{
		Options:     args.Options,
		ScanDirs:    scanDirs,
		ModuleSpecs: modSpecs,
		Logger:      args.Logger,
		Writer:      args.Writer,
	}
end:
	return c, err
}
