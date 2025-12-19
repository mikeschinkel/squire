package retinue

import (
	"log/slog"

	"github.com/mikeschinkel/go-cliutil"
	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/squire/squirepkg/squire"
)

var _ cliutil.Config = (*Config)(nil)

type Config struct {
	ScanDirs    []dt.DirPath
	ModuleSpecs []ModuleSpec
	Options     *squire.Options
	Logger      *slog.Logger
	Writer      cliutil.Writer
}

func (c *Config) Config() {}

type ConfigArgs struct {
	Options *squire.Options
	Logger  *slog.Logger
	Writer  cliutil.Writer
}
