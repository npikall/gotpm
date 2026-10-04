// Package logger builds the application logger for a given verbosity level.
package logger

import (
	"os"

	"charm.land/log/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/npikall/gotpm/internal/ui"
)

func Setup(level int) *log.Logger {
	// Writes pause the spinner, so a log line never lands on top of it. The
	// wrapper hides the terminal, so colors are detected on stderr itself.
	logger := log.New(ui.Pausing(os.Stderr))
	logger.SetColorProfile(colorprofile.Detect(os.Stderr, os.Environ()))
	logger.SetReportTimestamp(false)
	switch {
	case level >= 2: //nolint: mnd
		logger.SetLevel(log.DebugLevel)
	case level == 1:
		logger.SetLevel(log.InfoLevel)
	default:
		logger.SetLevel(log.WarnLevel)
	}
	return logger
}
