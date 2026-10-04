package locate

import (
	"io"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/npikall/gotpm/internal/ui"
)

const indent = "  "

func render(w io.Writer, groups []Group) {
	width := keyWidth(groups)
	for i, group := range groups {
		if i > 0 {
			_, _ = lipgloss.Fprintln(w)
		}
		_, _ = lipgloss.Fprintln(w, ui.Green.Render(group.Name))
		for _, entry := range group.Entries {
			renderEntry(w, entry, width)
		}
	}
}

func renderEntry(w io.Writer, entry Entry, width int) {
	key := ui.Normal.Render(entry.Key + strings.Repeat(" ", width-len(entry.Key)))
	if entry.Err != nil {
		_, _ = lipgloss.Fprintf(w, "%s%s %s\n", indent, key, ui.RedBold.Render("unresolved"))
		_, _ = lipgloss.Fprintf(w, "%s%s %s\n", indent, strings.Repeat(" ", width), ui.Muted.Render(entry.Err.Error()))
		return
	}
	line := ui.AccentBold.Render(entry.Path)
	if entry.Note != "" {
		line += " " + ui.Muted.Render("("+entry.Note+")")
	}
	_, _ = lipgloss.Fprintf(w, "%s%s %s\n", indent, key, line)
}

func keyWidth(groups []Group) int {
	width := 0
	for _, group := range groups {
		for _, entry := range group.Entries {
			if len(entry.Key) > width {
				width = len(entry.Key)
			}
		}
	}
	return width
}
