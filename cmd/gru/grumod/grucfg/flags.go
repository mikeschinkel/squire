package grucfg

import (
	"bytes"
	"errors"
	"flag"

	"github.com/mikeschinkel/go-cliutil"
	"github.com/mikeschinkel/squire/gru/grumod/gru"
)

type globalFlags = cliutil.GlobalFlags
type Flags struct {
	*globalFlags
	InputPath  string
	OutputPath string
	ModuleDir  string
}

func (*Flags) Flags() {}

func (flgs *Flags) GlobalFlags() *cliutil.GlobalFlags {
	return flgs.globalFlags
}

var _ cliutil.ArgsParser = (*Flags)(nil)

func NewFlags() *Flags {
	return &Flags{}
}

func (flgs *Flags) SetGlobalFlags(gFlags *cliutil.GlobalFlags) {
	flgs.globalFlags = gFlags
}

func (flgs *Flags) Parse(args []string) (err error) {
	var buf bytes.Buffer
	var errs []error

	fs := flag.NewFlagSet(string(gru.ExeName), flag.ContinueOnError)
	fs.SetOutput(&buf)

	fs.StringVar(&flgs.InputPath, "input", "", "Path to input JSON file (required)")
	fs.StringVar(&flgs.OutputPath, "output", "", "Path to output JSON file (required)")
	fs.StringVar(&flgs.ModuleDir, "module-dir", "", "Optional module directory override")

	err = fs.Parse(args)
	if err != nil {
		goto end
	}

	if buf.Len() != 0 {
		errs = AppendErr(errs, NewErr(
			ErrParsingFlags,
			errors.New(buf.String()),
		))
	}

	if len(errs) > 0 {
		err = CombineErrs(errs)
		goto end
	}

end:
	return err
}
