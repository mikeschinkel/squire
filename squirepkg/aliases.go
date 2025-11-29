package squirepkg

import (
	"context"

	"github.com/mikeschinkel/go-cliutil"
)

var stdErrf = cliutil.Stderrf

type Context context.Context

type CLIWriter = cliutil.Writer
