package internal

import (
	"os"

	"charm.land/log/v2"
	"github.com/npikall/gotpm/internal/ui"
	"github.com/spf13/cobra"
)

// SetupLogger creates a logger whose level is controlled by the --verbose flag.
func SetupLogger(cmd *cobra.Command) *log.Logger {
	logger := log.New(os.Stderr)
	logger.SetReportTimestamp(true)
	verboseCount, err := cmd.Flags().GetCount("verbose")
	if err != nil {
		logger.SetLevel(log.WarnLevel)
		return logger
	}
	switch {
	case verboseCount >= 2: //nolint: mnd
		logger.SetLevel(log.DebugLevel)
	case verboseCount == 1:
		logger.SetLevel(log.InfoLevel)
	default:
		logger.SetLevel(log.WarnLevel)
	}
	return logger
}

// Must returns t, or exits the process with a fatal log entry if err is non-nil.
func Must[T any](t T, err error) T { //nolint: ireturn
	if err != nil {
		ui.Error(err)
		os.Exit(1)
	}
	return t
}
