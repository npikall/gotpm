package ui

import (
	"fmt"

	"charm.land/lipgloss/v2"
)

// stdout is where the printers write: lipgloss's color-downsampling stdout,
// paused around any spinner showing.
var stdout = Pausing(lipgloss.Writer)

func Print(v ...any) {
	_, _ = fmt.Fprint(stdout, v...)
}

func Printf(format string, a ...any) {
	_, _ = fmt.Fprintf(stdout, format, a...)
}

func Infof(format string, a ...any) {
	prefix := Green.Render("info")
	text := Normal.Render(fmt.Sprintf(format, a...))
	_, _ = fmt.Fprintf(stdout, "%s: %s\n", prefix, text)
}

func Warnf(format string, a ...any) {
	prefix := YellowBold.Render("warning")
	text := Normal.Render(fmt.Sprintf(format, a...))
	_, _ = fmt.Fprintf(stdout, "%s: %s\n", prefix, text)
}

// Missingf reports something that could not be found.
func Missingf(format string, a ...any) {
	prefix := RedBold.Render("missing")
	text := Normal.Render(fmt.Sprintf(format, a...))
	_, _ = fmt.Fprintf(stdout, "%s: %s\n", prefix, text)
}

func Error(err error) {
	prefix := RedBold.Render("error")
	text := Normal.Render(err.Error())
	_, _ = fmt.Fprintf(stdout, "%s: %s\n", prefix, text)
}

// Package highlights a package reference, e.g. "@preview/my-pkg:0.1.0".
func Package(ref string) string {
	return AccentBold.Render(ref)
}
