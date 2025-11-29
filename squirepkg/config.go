package squirepkg

import (
	"github.com/mikeschinkel/squire/squirepkg/common"
	"github.com/mikeschinkel/squire/squirepkg/squirecfg"
)

func ParseConfig(cfg *squirecfg.RootConfigV1, args common.ConfigArgs) (c *common.Config, err error) {
	// TODO This will need to convert all these three source into a single central configuration
	c = &common.Config{
		Options: args.Options,
		AppInfo: args.AppInfo,
		Logger:  args.Logger,
		Writer:  args.Writer,
	}
	return c, err
}
