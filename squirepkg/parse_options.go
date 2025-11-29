package squirepkg

import (
	"github.com/mikeschinkel/go-cliutil"
	"github.com/mikeschinkel/squire/squirepkg/common"
	"github.com/mikeschinkel/squire/squirepkg/squirecfg"
)

// ParseOptions converts raw options from clicfg.Options into
// validated common.Options with embedded cliutil.GlobalOptions.
// If cliutilOpts is provided, it will be used directly instead of creating a new GlobalOptions.
func ParseOptions(cfgOpts *squirecfg.Options, cliutilOpts *cliutil.CLIOptions) (opts *common.Options, err error) {
	var cliOpts *cliutil.CLIOptions

	// If cliutilOpts provided, use it directly (preserves all flags like --force, --dry-run)
	if cliutilOpts != nil {
		cliOpts = cliutilOpts
	} else {
		// Otherwise create new GlobalOptions from config values
		cliOpts, err = cliutil.NewCLIOptions(cliutil.CLIOptionsArgs{
			Quiet:     &cfgOpts.Quiet,
			Verbosity: &cfgOpts.Verbosity,
		})
		if err != nil {
			goto end
		}
	}

	opts = common.NewOptions(common.OptionsArgs{
		CLIOptions: cliOpts,
	})

end:
	return opts, err
}
