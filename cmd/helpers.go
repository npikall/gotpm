package cmd

import (
	"fmt"
	"log"

	"github.com/charmbracelet/lipgloss"
)

// Common Colors
var (
	Cyan   = lipgloss.Color("81")
	Yellow = lipgloss.Color("3")
	Red    = lipgloss.Color("124")
)

// Common Styles
var (
	InfoStyle = lipgloss.NewStyle().Foreground(Cyan).Bold(true)
	WarnStyle = lipgloss.NewStyle().Foreground(Yellow).Bold(true)
	ErrStyle  = lipgloss.NewStyle().Foreground(Red).Bold(true)
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

// A given Function must return no error.
// When an error occurs the program is exited.
func Must[T any](t T, err error) T {
	if err != nil {
		log.Fatalf("%s: %s", ErrStyle.Render("error"), err)
	}
	return t
}
