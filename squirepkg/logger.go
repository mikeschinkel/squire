package squirepkg

import (
	"log/slog"

	"github.com/mikeschinkel/squire/squirepkg/common"
)

// logger is the package-level logger instance.
var logger *slog.Logger

func init() {
	common.RegisterSetLoggerFunc(func(l *slog.Logger) {
		logger = l
	})
}
