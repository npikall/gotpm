package locate

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/npikall/gotpm/internal/ui"
)

const indent = "  "

// render prints the groups, keys padded to the widest key in the whole dump so
// the paths line up across group boundaries rather than only within a group.
func render(groups []Group) {
	width := keyWidth(groups)
	for i, group := range groups {
		if i > 0 {
			_, _ = lipgloss.Println()
		}
		_, _ = lipgloss.Println(ui.Green.Render(group.Name))
		for _, entry := range group.Entries {
			renderEntry(entry, width)
		}
	}
}

func renderEntry(entry Entry, width int) {
	key := ui.Normal.Render(entry.Key + strings.Repeat(" ", width-len(entry.Key)))
	if entry.Err != nil {
		_, _ = lipgloss.Printf("%s%s %s\n", indent, key, ui.RedBold.Render("unresolved"))
		_, _ = lipgloss.Printf("%s%s %s\n", indent, strings.Repeat(" ", width), ui.Muted.Render(entry.Err.Error()))
		return
	}
	// %s, not %q: the path is styled, and quoting would escape the ANSI codes.
	line := ui.AccentBold.Render(entry.Path)
	if entry.Note != "" {
		line += " " + ui.Muted.Render("("+entry.Note+")")
	}
	_, _ = lipgloss.Printf("%s%s %s\n", indent, key, line)
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
