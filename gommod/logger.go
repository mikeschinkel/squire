package gommod

import (
	"log/slog"

	"github.com/mikeschinkel/gomion/gommod/gomion"
)

// logger is the package-level logger instance.
var logger *slog.Logger

func init() {
	gomion.RegisterSetLoggerFunc(func(l *slog.Logger) {
		logger = l
	})
}
