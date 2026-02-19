package cmd

import (
	"os"

	"github.com/charmbracelet/lipgloss/v2"
	"github.com/charmbracelet/log"
	"github.com/charmbracelet/x/exp/charmtone"
	"github.com/spf13/cobra"
)

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
