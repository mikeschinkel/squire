package squirepkg

import (
	"github.com/mikeschinkel/squire/squirepkg/squire"
	"github.com/mikeschinkel/squire/squirepkg/squirecfg"
)

func ParseConfig(cfg *squirecfg.RootConfigV1, args squire.ConfigArgs) (c *squire.Config, err error) {
	// TODO This will need to convert all these three source into a single central configuration
	c = &squire.Config{
		Options: args.Options,
		AppInfo: args.AppInfo,
		Logger:  args.Logger,
		Writer:  args.Writer,
	}
	return c, err
}
