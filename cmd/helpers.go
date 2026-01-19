package cmd

import (
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
)

// Common Colors
var (
	Blue    = lipgloss.Color("81")
	Yellow  = lipgloss.Color("3")
	Red     = lipgloss.Color("124")
	White   = lipgloss.Color("231")
	Gray    = lipgloss.Color("245")
	Magenta = lipgloss.Color("72")
	Cyan    = lipgloss.Color("117")
	Violet  = lipgloss.Color("99")
)

// Styles for Messages to stdout/stderr
var (
	InfoStyle    = lipgloss.NewStyle().Foreground(Blue).Bold(true)
	WarnStyle    = lipgloss.NewStyle().Foreground(Yellow).Bold(true)
	ErrStyle     = lipgloss.NewStyle().Foreground(Red).Bold(true)
	HighStyle    = lipgloss.NewStyle().Foreground(White).Bold(true)
	DefaultStyle = lipgloss.NewStyle().Foreground(Gray)
	LogoStyle    = lipgloss.NewStyle().Foreground(Violet)
)

// Styles for the List Command
var (
	namespaceStyle = lipgloss.NewStyle().Bold(true).Foreground(Magenta).MarginTop(1)
	packageStyle   = lipgloss.NewStyle().Bold(true).Foreground(Cyan)
	versionStyle   = lipgloss.NewStyle().Foreground(Gray)
	countStyle     = lipgloss.NewStyle().Foreground(Gray)
)

func setupLogger() *log.Logger {
	logger := log.New(os.Stdout)
	logger.SetReportCaller(false)
	logger.SetReportTimestamp(false)
	logger.SetLevel(log.InfoLevel)
	return logger
}
func setupVerboseLogger(cmd *cobra.Command) *log.Logger {
	verbose := Must(cmd.Flags().GetBool("debug"))
	logger := setupLogger()
	if verbose {
		logger.SetLevel(log.DebugLevel)
	}
	return logger
}

// A given Function must return no error.
// When an error occurs the program is exited.
func Must[T any](t T, err error) T {
	if err != nil {
		log.Fatal(err)
	}
	return t
}
