package cmd

import (
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
)

// Common Colors
var (
	Cyan   = lipgloss.Color("81")
	Yellow = lipgloss.Color("3")
	Red    = lipgloss.Color("124")
	White  = lipgloss.Color("231")
)

// Common Styles
var (
	InfoStyle = lipgloss.NewStyle().Foreground(Cyan).Bold(true)
	WarnStyle = lipgloss.NewStyle().Foreground(Yellow).Bold(true)
	ErrStyle  = lipgloss.NewStyle().Foreground(Red).Bold(true)
	HighStyle = lipgloss.NewStyle().Foreground(White).Bold(true)
)

// Print a message to stdout with colored prefix 'info:'
func LogInfof(format string, v ...any) {
	msg := fmt.Sprintf(format, v...)
	fmt.Printf("%s: %s\n", InfoStyle.Render("info"), msg)
}

// Print a warning message to stdout with colored prefix 'warning:'
func LogWarnf(format string, v ...any) {
	msg := fmt.Sprintf(format, v...)
	fmt.Printf("%s: %s\n", WarnStyle.Render("warning"), msg)
}

// Print an error message to stdout with colored prefix 'error:' without exiting
func LogErrf(format string, v ...any) {
	msg := fmt.Sprintf(format, v...)
	fmt.Printf("%s: %s\n", ErrStyle.Render("error"), msg)
}

// Print an error message to stdout with colored prefix 'error:' without exiting
func LogFatalf(format string, v ...any) {
	msg := fmt.Sprintf(format, v...)
	fmt.Printf("%s: %s\n", ErrStyle.Render("error"), msg)
	os.Exit(1)
}

// A given Function must return no error.
// When an error occurs the program is exited.
func Must[T any](t T, err error) T {
	if err != nil {
		fmt.Printf("%s: %s\n", ErrStyle.Render("error"), err)
		os.Exit(1)
	}
	return t
}
