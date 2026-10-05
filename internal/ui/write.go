package ui

import (
	"fmt"
	"io"
	"os"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"
)

// stdout is where the printers write: lipgloss's color-downsampling stdout,
// paused around any spinner showing.
var stdout = Pausing(lipgloss.Writer)

// Stderr returns stderr, downsampling colors to what it supports and paused
// around any spinner showing.
func Stderr() io.Writer {
	return Pausing(colorprofile.NewWriter(os.Stderr, os.Environ()))
}

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

// Command renders cmd as a shell prompt, e.g. "$ rm -f thumbnail.png".
func Command(cmd string) string {
	return Green.Render("$") + " " + AccentBold.Render(cmd)
}

// Notes prints info and then warning, skipping whichever is empty.
func Notes(info, warning string) {
	if info != "" {
		Infof("%s", info)
	}
	if warning != "" {
		Warnf("%s", warning)
	}
}
