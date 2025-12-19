package squirepkg

import (
	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/squire/squirepkg/retinue"
	"github.com/mikeschinkel/squire/squirepkg/squirecfg"
)

func ParseConfig(cfg *squirecfg.RootConfigV1, args retinue.ConfigArgs) (c *retinue.Config, err error) {
	var scanDirs []dt.DirPath
	var modSpecs []retinue.ModuleSpec

	scanDirs, err = dt.ParseDirPaths(cfg.ScanDirs)
	if err != nil {
		goto end
	}

	modSpecs, err = retinue.ParseModuleSpecs(cfg.ModuleSpecs)
	if err != nil {
		goto end
	}

	c = &retinue.Config{
		Options:     args.Options,
		ScanDirs:    scanDirs,
		ModuleSpecs: modSpecs,
		Logger:      args.Logger,
		Writer:      args.Writer,
	}
end:
	return c, err
}
