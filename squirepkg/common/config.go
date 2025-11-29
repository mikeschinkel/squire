package common

import (
	"log/slog"

	"github.com/mikeschinkel/go-cliutil"
	"github.com/mikeschinkel/go-dt/appinfo"
)

// Singleton instance for CLI command configuration
var config *Config

var _ cliutil.Config = (*Config)(nil)

type Config struct {
	Options *Options
	AppInfo appinfo.AppInfo
	Logger  *slog.Logger
	Writer  cliutil.Writer
}

// GetConfig returns the singleton config instance
func GetConfig() *Config {
	if config == nil {
		panic("common.Initialize() must be called before calling common.GetConfig()")
	}
	return config
}

func (c *Config) Config() {}

type ConfigArgs struct {
	Options *Options
	AppInfo appinfo.AppInfo
	Logger  *slog.Logger
	Writer  cliutil.Writer
}

func ParseConfig(args ConfigArgs) (c *Config, err error) {
	// TODO This will need to convert all these three source into a single central configuration
	c = &Config{
		Options: args.Options,
		AppInfo: args.AppInfo,
		Logger:  args.Logger,
		Writer:  args.Writer,
	}
	return c, err
}
