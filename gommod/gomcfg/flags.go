package gomcfg

import (
	"bytes"
	"errors"
	"flag"

	"github.com/mikeschinkel/go-cliutil"
	"github.com/mikeschinkel/gomion/gommod/gomion"
)

type globalFlags = cliutil.GlobalFlags
type Flags struct {
	*globalFlags
	InputPath  string
	OutputPath string
	// ModuleDir is provided as the first positional argument (default ".").
	ModuleDir string
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

	fs := flag.NewFlagSet(string(gomion.ExeName), flag.ContinueOnError)
	fs.SetOutput(&buf)

	fs.StringVar(&flgs.InputPath, "input", "", "RelPath to input JSON file (required)")
	fs.StringVar(&flgs.OutputPath, "output", "", "RelPath to output JSON file (required)")
	// fs.StringVar(&flgs.ModuleDir, "module-dir", "", "Optional module directory override") // disabled: module dir is positional

	err = fs.Parse(args)
	if err != nil {
		goto end
	}

	// First positional argument sets the module directory; default "." when not provided.
	flgs.ModuleDir = "."
	if fs.NArg() > 0 {
		flgs.ModuleDir = fs.Arg(0)
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
