package cmd

import (
	"os"
	"time"

	"github.com/briandowns/spinner"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/charmbracelet/log"
	"github.com/charmbracelet/x/exp/charmtone"
	"github.com/spf13/cobra"
)

// Helper to log in GoRoutines
type logEvent struct {
	level   string
	msg     string
	keyvals []any
}

// Common Colors
var (
	Violet    = charmtone.Charple
	Turquoise = lipgloss.Color("86")
	Magenta   = charmtone.Cheeky
)

// Styles for Messages to stdout/stderr
var (
	VersionStyle     = lipgloss.NewStyle().Foreground(Turquoise).Bold(true)
	HighStyle        = lipgloss.NewStyle().Foreground(Magenta).Bold(true)
	LogoStyle        = lipgloss.NewStyle().Foreground(Violet)
	DescriptionStyle = lipgloss.NewStyle().Foreground(Magenta)
)

// Styles for the List Command
var (
	namespaceStyle = lipgloss.NewStyle().Foreground(Violet).MarginTop(1)
	packageStyle   = lipgloss.NewStyle().Foreground(Magenta)
	versionStyle   = lipgloss.NewStyle().Faint(true)
	countStyle     = lipgloss.NewStyle().Faint(true)
)

func setupLogger(cmd *cobra.Command) *log.Logger {
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

func setupSpinner() *spinner.Spinner {
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = countStyle.Render(" Loading...")
	_ = s.Color("cyan")
	return s
}

// A given Function must return no error.
// When an error occurs the program is exited.
func Must[T any](t T, err error) T {
	if err != nil {
		log.Fatal(err)
	}
	return t
}
