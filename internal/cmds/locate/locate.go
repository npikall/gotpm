// Package locate implements the locate command: it reports where installed
// packages are stored.
package locate

import (
	"fmt"

	"charm.land/log/v2"
	"github.com/npikall/gotpm/internal/paths"
	"github.com/npikall/gotpm/internal/ui"
)

// Run prints the directory typst packages are installed into, creating it if
// it does not exist yet.
func Run(log *log.Logger) error {
	target, err := paths.EnsureTypstPackagesDir()
	if err != nil {
		return fmt.Errorf("could not resolve package directory: %w", err)
	}
	log.Debug("resolved", "path", target)
	// %s, not %q: the path is styled, and quoting would escape the ANSI codes.
	ui.Infof(`packages located at "%s"`, ui.AccentBold.Render(target))
	return nil
}
