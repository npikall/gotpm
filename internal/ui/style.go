// Package ui owns everything gotpm writes to a terminal: colors, message
// prefixes and the spinner. No other internal package prints.
package ui

import (
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/exp/charmtone"
)

var (
	blue      = charmtone.Malibu
	green     = charmtone.Bok
	yellow    = charmtone.Zest
	red       = lipgloss.Red
	violet    = charmtone.Charple
	magenta   = charmtone.Cheeky
	normal    = lipgloss.White
	muted     = charmtone.Squid
	accent    = charmtone.Sash
	ansigreen = lipgloss.Green
)

var (
	BlueBold      = lipgloss.NewStyle().Foreground(blue).Bold(true)
	Blue          = lipgloss.NewStyle().Foreground(blue)
	Green         = lipgloss.NewStyle().Foreground(green)
	ANSIGreenBold = lipgloss.NewStyle().Foreground(ansigreen).Bold(true)
	YellowBold    = lipgloss.NewStyle().Foreground(yellow).Bold(true)
	Red           = lipgloss.NewStyle().Foreground(red)
	RedBold       = lipgloss.NewStyle().Foreground(red).Bold(true)
	Normal        = lipgloss.NewStyle().Foreground(normal)
	Muted         = lipgloss.NewStyle().Foreground(muted)
	AccentBold    = lipgloss.NewStyle().Foreground(accent).Bold(true)

	Logo        = lipgloss.NewStyle().Foreground(violet)
	Description = lipgloss.NewStyle().Foreground(magenta)
)
