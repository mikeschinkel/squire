package gompkg

import (
	"log/slog"

	"github.com/mikeschinkel/go-cliutil"
	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/gomion/gommod/gomion"
)

var _ cliutil.Config = (*Config)(nil)

type Config struct {
	ScanDirs    []dt.DirPath
	ModuleSpecs []ModuleSpec
	Options     *gomion.Options
	Logger      *slog.Logger
	Writer      cliutil.Writer
}

func (c *Config) Config() {}

type ConfigArgs struct {
	Options *gomion.Options
	Logger  *slog.Logger
	Writer  cliutil.Writer
}
