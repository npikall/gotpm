package internal

import (
	"os"
	"time"

	"github.com/briandowns/spinner"
	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
)

// SetupLogger creates a logger whose level is controlled by the --verbose flag.
func SetupLogger(cmd *cobra.Command) *log.Logger {
	logger := log.New(os.Stdout)
	logger.SetReportTimestamp(true)
	verboseCount, err := cmd.Flags().GetCount("verbose")
	if err != nil {
		logger.SetLevel(log.WarnLevel)
		return logger
	}
	switch {
	case verboseCount >= 2:
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
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = StyleMuted.Render(" Loading...")
	_ = s.Color("cyan")
	return s
}

// Must returns t, or exits the process with a fatal log entry if err is non-nil.
func Must[T any](t T, err error) T {
	if err != nil {
		log.Fatal(err)
	}
	return t
}
