package internal

import (
	"fmt"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/exp/charmtone"
)

var (
	Blue      = charmtone.Malibu
	Green     = charmtone.Bok
	Yellow    = charmtone.Zest
	Red       = charmtone.Coral
	Violet    = charmtone.Charple
	Magenta   = charmtone.Cheeky
	Normal    = lipgloss.White
	Muted     = charmtone.Squid
	Accent    = charmtone.Ash
	ANSIGreen = lipgloss.Green
)

var (
	StyleBlueBold    = lipgloss.NewStyle().Foreground(Blue).Bold(true)
	StyleBlue        = lipgloss.NewStyle().Foreground(Blue)
	StyleGreen       = lipgloss.NewStyle().Foreground(Green)
	StyleANSIGreen   = lipgloss.NewStyle().Foreground(ANSIGreen).Bold(true)
	StyleYellow      = lipgloss.NewStyle().Foreground(Yellow).Bold(true)
	StyleRed         = lipgloss.NewStyle().Foreground(Red)
	StyleNormal      = lipgloss.NewStyle().Foreground(Normal)
	StyleMuted       = lipgloss.NewStyle().Foreground(Muted)
	StyleAccent      = lipgloss.NewStyle().Foreground(Accent).Bold(true)
	StyleLogo        = lipgloss.NewStyle().Foreground(Violet)
	StyleDescription = lipgloss.NewStyle().Foreground(Magenta)
)

func PrintInfof(format string, a ...any) {
	prefix := StyleBlueBold.Render("info")
	text := StyleNormal.Render(fmt.Sprintf(format, a...))
	_, _ = lipgloss.Printf("%s: %s\n", prefix, text)
}

func PrintWarnf(format string, a ...any) {
	prefix := StyleYellow.Render("warning")
	text := StyleAccent.Render(fmt.Sprintf(format, a...))
	_, _ = lipgloss.Printf("%s: %s\n", prefix, text)
}

func FormatImportStmt(namespace, name, version string) string {
	return StyleAccent.Render(fmt.Sprintf("@%s/%s:%s", namespace, name, version))
}
