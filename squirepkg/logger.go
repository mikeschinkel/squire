package squirepkg

import (
	"log/slog"

	"github.com/mikeschinkel/squire/squirepkg/squire"
)

// logger is the package-level logger instance.
var logger *slog.Logger

func init() {
	squire.RegisterSetLoggerFunc(func(l *slog.Logger) {
		logger = l
	})
}
