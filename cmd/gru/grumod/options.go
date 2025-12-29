package grumod

import (
	"errors"

	"github.com/mikeschinkel/go-cliutil"
	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/go-dt/dtx"
	"github.com/mikeschinkel/go-jsonxtractr"
	"github.com/mikeschinkel/squire/gru/grumod/gru"
	"github.com/mikeschinkel/squire/gru/grumod/grucfg"
	"github.com/mikeschinkel/squire/gru/grumod/grupkg"
)

var (
	ErrInvalidInputPath  = errors.New("invalid --input path")
	ErrInvalidOutputPath = errors.New("invalid --output path")
	ErrInvalidModuleDir  = errors.New("invalid --module-dir")
	ErrInvalidInputFile  = errors.New("invalid input file")
)

var _ cliutil.Options = (*Options)(nil)

type Options struct {
	*cliutil.GlobalFlags
	InputPath  dt.Filepath
	OutputPath dt.Filepath
	ModuleDir  dt.DirPath
	InputData  grupkg.InputData
}

func NewOptions() *Options {
	return &Options{}
}

func (*Options) Options() {}

func (opts *Options) Parse(flags cliutil.Flags) error {
	var inputData grucfg.InputData

	gruFlags, err := dtx.AssertType[*grucfg.Flags](flags)
	if err != nil {
		panic(err.Error())
	}

	opts.GlobalFlags = gruFlags.GlobalFlags()

	opts.InputPath, err = dt.EnsureFilepath(gruFlags.InputPath, gru.DefaultInputFile)
	if err != nil {
		err = NewErr(ErrInvalidInputPath, err)
		goto end
	}

	opts.OutputPath, err = dt.EnsureFilepath(gruFlags.OutputPath, gru.DefaultOutputFile)
	if err != nil && !errors.Is(err, dt.ErrFileNotExists) {
		err = NewErr(ErrInvalidOutputPath, err)
		goto end
	}

	opts.ModuleDir, err = dt.ParseDirPath(gruFlags.ModuleDir)
	if err != nil && !errors.Is(err, dt.ErrEmpty) {
		err = NewErr(ErrInvalidModuleDir, opts.ModuleDir.ErrKV(), err)
		goto end
	}

	err = jsonxtractr.Load(opts.InputPath, &inputData)
	if err != nil {
		goto end
	}

	opts.InputData, err = grupkg.ParseInputData(inputData)
	if err != nil {
		err = NewErr(ErrInvalidInputFile, opts.InputPath.ErrKV(), err)
		goto end
	}

end:
	return err
}
