package ui

import (
	"fmt"

	"charm.land/lipgloss/v2"
)

func Print(v ...any) {
	_, _ = lipgloss.Print(v...)
}

func Printf(format string, a ...any) {
	_, _ = lipgloss.Printf(format, a...)
}

func Infof(format string, a ...any) {
	prefix := Green.Render("info")
	text := Normal.Render(fmt.Sprintf(format, a...))
	_, _ = lipgloss.Printf("%s: %s\n", prefix, text)
}

func Warnf(format string, a ...any) {
	prefix := YellowBold.Render("warning")
	text := Normal.Render(fmt.Sprintf(format, a...))
	_, _ = lipgloss.Printf("%s: %s\n", prefix, text)
}

// Missingf reports something that could not be found.
func Missingf(format string, a ...any) {
	prefix := RedBold.Render("missing")
	text := Normal.Render(fmt.Sprintf(format, a...))
	_, _ = lipgloss.Printf("%s: %s\n", prefix, text)
}

func Error(err error) {
	prefix := RedBold.Render("error")
	text := Normal.Render(err.Error())
	_, _ = lipgloss.Printf("%s: %s\n", prefix, text)
}

// Package highlights a package reference, e.g. "@preview/my-pkg:0.1.0".
func Package(ref string) string {
	return AccentBold.Render(ref)
}
