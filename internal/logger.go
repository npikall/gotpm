package internal

import (
	"os"
	"time"

	"charm.land/log/v2"
	"github.com/briandowns/spinner"
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

// SetupSpinner returns a spinner ready to start.
func SetupSpinner() *spinner.Spinner {
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond) //nolint: mnd
	s.Suffix = StyleMuted.Render(" Loading...")
	_ = s.Color("cyan")
	return s
}

// Must returns t, or exits the process with a fatal log entry if err is non-nil.
func Must[T any](t T, err error) T { //nolint: ireturn
	if err != nil {
		PrintError(err)
		os.Exit(1)
	}
	return t
}
