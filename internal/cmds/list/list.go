// Package list implements the list command: it shows every package installed
// in the local package directory.
package list

import (
	"errors"
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"charm.land/log/v2"
	"github.com/npikall/gotpm/internal/store"
	"github.com/npikall/gotpm/internal/ui"
)

var ErrNoPackages = errors.New("no packages installed")

// maxDisplayedVersions caps how many versions of a package are spelled out
// before the rest are summarised.
const maxDisplayedVersions = 5

// Run prints every installed package, grouped by namespace.
func Run(log *log.Logger) error {
	s, err := store.OpenPackageDir()
	if err != nil {
		return err
	}
	log.Debug("looking in", "directory", s.Root())

	if !s.Exists() {
		return ErrNoPackages
	}

	namespaces, err := s.Scan()
	if err != nil {
		return err
	}
	if len(namespaces) == 0 {
		log.Info("no packages found")
		return nil
	}

	render(namespaces)
	return nil
}

func render(namespaces []store.Namespace) {
	total := 0
	for _, namespace := range namespaces {
		_, _ = lipgloss.Println(ui.Green.Render("@" + namespace.Name))
		for _, p := range namespace.Packages {
			total++
			renderPackage(p)
		}
	}

	footer := fmt.Sprintf("Total: %d packages across %d namespaces", total, len(namespaces))
	_, _ = lipgloss.Println()
	_, _ = lipgloss.Println(ui.Muted.Render(footer))
}

func renderPackage(p store.Package) {
	versions := p.Versions
	truncated := ""
	if len(versions) > maxDisplayedVersions {
		truncated = fmt.Sprintf(" ... (+%d more)", len(versions)-maxDisplayedVersions)
		versions = versions[:maxDisplayedVersions]
	}

	parts := make([]string, 0, len(versions))
	for _, version := range versions {
		if version.Editable {
			parts = append(parts, ui.YellowBold.Render(version.Name+" (editable)"))
		} else {
			parts = append(parts, ui.Muted.Render(version.Name))
		}
	}

	_, _ = lipgloss.Printf("  %s %s%s\n",
		ui.Normal.Render(p.Name),
		strings.Join(parts, ui.Muted.Render(", ")),
		ui.Muted.Render(truncated),
	)
}
