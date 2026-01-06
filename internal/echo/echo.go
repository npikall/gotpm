package echo

import (
	"fmt"
	"os"

	"github.com/fatih/color"
)

var (
	red    = color.New(color.FgHiRed, color.Bold).SprintFunc()
	yellow = color.New(color.FgYellow, color.Bold).SprintFunc()
	green  = color.New(color.FgHiGreen, color.Bold).SprintFunc()
)

// Print format an error message to stderr with colored prefix 'error:'
func EchoErrorf(format string, a ...any) {
	msg := fmt.Sprintf(format, a...)
	fmt.Fprintf(os.Stderr, "%s: %s\n", red("error"), msg)
}

// Print an error message to stderr with colored prefix 'error:'
func EchoError(msg string) {
	fmt.Fprintf(os.Stderr, "%s: %s\n", red("error"), msg)
}

// Print a colored error message and exit program
func ExitErrorf(format string, a ...any) {
	EchoErrorf(format, a...)
	os.Exit(1)
}

// Print a colored error message and exit program
func ExitError(msg string) {
	EchoError(msg)
	os.Exit(1)
}

// Print format a warning message to stdout with colored prefix 'warning:'
func EchoWarningf(format string, a ...any) {
	msg := fmt.Sprintf(format, a...)
	fmt.Fprintf(os.Stdout, "%s: %s\n", yellow("warning"), msg)
}

// Print a warning message to stdout with colored prefix 'warning:'
func EchoWarning(msg string) {
	fmt.Fprintf(os.Stdout, "%s: %s\n", yellow("warning"), msg)
}

// Print a warning message to stdout with colored prefix 'info:'
func EchoInfof(format string, a ...any) {
	msg := fmt.Sprintf(format, a...)
	fmt.Fprintf(os.Stdout, "%s: %s\n", green("info"), msg)
}

// Print a warning message to stdout with colored prefix 'info:'
func EchoInfo(msg string) {
	fmt.Fprintf(os.Stdout, "%s: %s\n", green("info"), msg)
}
