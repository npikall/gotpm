package compilers

import (
	"fmt"
)

// Selection is the compiler a project is compiled with.
type Selection struct {
	Version string
	// NewPin reports that Version is newly chosen and has to be written as
	// the project's compiler pin.
	NewPin bool
}

// Select chooses the compiler for a floor, the lowest compiler that may be
// used, and a pin, the compiler the project already pins. Either may be "".
//
// A pin satisfying the floor is kept. Otherwise the newest installed stable
// compiler satisfying the floor is chosen, else the latest stable release
// satisfying it; releases, the stable releases, is only asked for then.
// Pre-releases are never chosen, only kept when pinned.
func Select(floor, pin string, installed []string, releases func() ([]Release, error)) (Selection, error) {
	if err := validUnlessEmpty(floor); err != nil {
		return Selection{}, err
	}
	if err := validUnlessEmpty(pin); err != nil {
		return Selection{}, err
	}
	if pin != "" && satisfies(pin, floor) {
		return Selection{Version: pin}, nil
	}
	if newest := newestSatisfying(installed, floor); newest != "" {
		return Selection{Version: newest, NewPin: true}, nil
	}

	published, err := releases()
	if err != nil {
		return Selection{}, err
	}
	versions := make([]string, 0, len(published))
	for _, release := range published {
		versions = append(versions, release.Version)
	}
	if newest := newestSatisfying(versions, floor); newest != "" {
		return Selection{Version: newest, NewPin: true}, nil
	}
	if floor == "" {
		return Selection{}, fmt.Errorf("%w: no stable release published", ErrNoRelease)
	}
	return Selection{}, fmt.Errorf("%w: no stable release satisfies the compiler requirement %s", ErrNoRelease, floor)
}

// satisfies reports whether version is at least floor; every version
// satisfies no floor.
func satisfies(version, floor string) bool {
	return floor == "" || CompareVersions(version, floor) >= 0
}

// newestSatisfying is the newest stable version satisfying floor, or "".
func newestSatisfying(versions []string, floor string) string {
	var candidates []string
	for _, version := range versions {
		if validVersion(version) == nil && satisfies(version, floor) {
			candidates = append(candidates, version)
		}
	}
	return NewestStable(candidates)
}

// validUnlessEmpty checks version unless it is "", which stands for none.
func validUnlessEmpty(version string) error {
	if version == "" {
		return nil
	}
	return validVersion(version)
}
