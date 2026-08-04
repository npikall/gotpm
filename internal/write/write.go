package write

import (
	"fmt"

	"charm.land/lipgloss/v2"
	"github.com/npikall/gotpm/internal/style"
)

func Infof(format string, a ...any) {
	prefix := style.BlueBold.Render("info")
	text := style.Normal.Render(fmt.Sprintf(format, a...))
	_, _ = lipgloss.Printf("%s: %s\n", prefix, text)
}

func Warnf(format string, a ...any) {
	prefix := style.YellowBold.Render("warning")
	text := style.Normal.Render(fmt.Sprintf(format, a...))
	_, _ = lipgloss.Printf("%s: %s\n", prefix, text)
}

func Error(msg string) {
	prefix := style.RedBold.Render("error")
	text := style.Normal.Render(msg)
	_, _ = lipgloss.Printf("%s: %s\n", prefix, text)
}
