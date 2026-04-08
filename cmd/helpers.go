package cmd

import (
	"fmt"
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

var (
	Blue    = charmtone.Malibu // Sardine
	Green   = charmtone.Bok    // Guac, Julep
	Yellow  = charmtone.Zest   // Citron, Mustard
	Red     = charmtone.Coral  // Sriracha, Chili
	Violet  = charmtone.Charple
	Magenta = charmtone.Cheeky
	Normal  = charmtone.Smoke // White
	Muted   = charmtone.Squid // Darker White
	Accent  = charmtone.Ash   // Brighter White
)

var (
	StyleBlueBold    = lipgloss.NewStyle().Foreground(Blue).Bold(true)
	StyleBlue        = lipgloss.NewStyle().Foreground(Blue)
	StyleGreen       = lipgloss.NewStyle().Foreground(Green)
	StyleYellow      = lipgloss.NewStyle().Foreground(Yellow)
	StyleRed         = lipgloss.NewStyle().Foreground(Red)
	StyleNormal      = lipgloss.NewStyle().Foreground(Normal)
	StyleMuted       = lipgloss.NewStyle().Foreground(Muted)
	StyleAccent      = lipgloss.NewStyle().Foreground(Accent).Bold(true)
	StyleLogo        = lipgloss.NewStyle().Foreground(Violet)
	StyleDescription = lipgloss.NewStyle().Foreground(Magenta)
)

func printInfo(format string, a ...any) {
	prefix := StyleBlueBold.Render("info")
	text := StyleNormal.Render(fmt.Sprintf(format, a...))
	fmt.Printf("%s: %s\n", prefix, text)
}

func printWarn(format string, a ...any) {
	prefix := StyleBlueBold.Render("warning")
	text := StyleNormal.Render(fmt.Sprintf(format, a...))
	fmt.Printf("%s: %s\n", prefix, text)
}

func formatImportStmt(namespace, name, version string) string {
	return StyleAccent.Render(fmt.Sprintf("@%s/%s:%s", namespace, name, version))
}

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
	s.Suffix = StyleMuted.Render(" Loading...")
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
