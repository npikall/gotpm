package cmd

import (
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
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

func setupLogger(verbose bool) *log.Logger {
	logger := log.New(os.Stdout)
	logger.SetReportCaller(false)
	logger.SetReportTimestamp(false)
	if verbose {
		logger.SetLevel(log.DebugLevel)
	} else {
		logger.SetLevel(log.InfoLevel)
	}
	return logger
}

// Print a message to stdout with colored prefix 'info:'
func LogInfof(format string, v ...any) {
	msg := fmt.Sprintf(format, v...)
	fmt.Printf("%s: %s\n", InfoStyle.Render("info"), DefaultStyle.Render(msg))
}

// Print a warning message to stdout with colored prefix 'warning:'
func LogWarnf(format string, v ...any) {
	msg := fmt.Sprintf(format, v...)
	fmt.Printf("%s: %s\n", WarnStyle.Render("warning"), DefaultStyle.Render(msg))
}

// Print an error message to stdout with colored prefix 'error:' without exiting
func LogErrf(format string, v ...any) {
	msg := fmt.Sprintf(format, v...)
	fmt.Printf("%s: %s\n", ErrStyle.Render("error"), DefaultStyle.Render(msg))
}

// Print an error message to stdout with colored prefix 'error:' without exiting
func LogFatalf(format string, v ...any) {
	msg := fmt.Sprintf(format, v...)
	fmt.Printf("%s: %s\n", ErrStyle.Render("error"), DefaultStyle.Render(msg))
	os.Exit(1)
}

// A given Function must return no error.
// When an error occurs the program is exited.
func Must[T any](t T, err error) T {
	if err != nil {
		log.Fatal(err)
	}
	return t
}
