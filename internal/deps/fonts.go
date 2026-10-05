package deps

import (
	"context"

	"charm.land/log/v2"
	"github.com/npikall/gotpm/internal/fonts"
	"github.com/npikall/gotpm/internal/lockfile"
	"github.com/npikall/gotpm/internal/ui"
)

// FontResults is what installing a set of font pins did, together with the
// font directory it did it in.
type FontResults struct {
	Dir     fonts.Dir
	Results []fonts.Result
}

// EnsureFonts makes the font directory hold every font pin. A family gotpm
// did not install is skipped unless forced, and reported by Report.
func EnsureFonts(pins []lockfile.Font, force bool, logger *log.Logger) (FontResults, error) {
	if len(pins) == 0 {
		return FontResults{}, nil
	}
	dir, err := fonts.OpenDir()
	if err != nil {
		return FontResults{}, err
	}
	logger.Debug("installing fonts", "dir", dir.Root, "count", len(pins))
	results, err := ui.WithSpinner(" installing fonts", func() ([]fonts.Result, error) {
		return dir.EnsureAll(context.Background(), pins, force)
	})
	return FontResults{Dir: dir, Results: results}, err
}

// Report tells what happened to each font that changed, and returns how many
// did.
func (r FontResults) Report() int {
	changed := 0
	for _, result := range r.Results {
		if result.Outcome == fonts.Installed || result.Outcome == fonts.Replaced {
			ui.Infof("installed font %q", result.Pin.Family)
			changed++
		}
		ui.Notes(result.Notes(r.Dir))
	}
	return changed
}
