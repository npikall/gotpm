package internal

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Version is a semantic version where only positive integers are allowed.
type Version struct {
	Major uint64
	Minor uint64
	Patch uint64
}

const (
	MAJOR string = "major"
	MINOR string = "minor"
	PATCH string = "patch"
)

var ErrInvalidIncrement = errors.New("invalid version incrementation, must be one of [major|minor|patch]")
var ErrInvalidVersion = errors.New("not a valid semantic version")

// Bump increments the Version by the given increment (major, minor, patch).
// Returns ErrInvalidIncrement if an unrecognized increment is used.
func (v *Version) Bump(increment string) error {
	switch increment {
	case MAJOR:
		v.Major += 1
		v.Minor = 0
		v.Patch = 0
	case MINOR:
		v.Minor += 1
		v.Patch = 0
	case PATCH:
		v.Patch += 1
	default:
		return ErrInvalidIncrement
	}
	return nil
}

func (v Version) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

// NewVersion returns a Version with all fields set to zero.
func NewVersion() Version {
	return Version{Major: 0, Minor: 0, Patch: 0}
}

// CompareVersions compares two Version structs. Useful for sorting.
// Returns 1 if a > b, -1 if a < b, 0 if equal.
func CompareVersions(a, b Version) int {
	switch {
	case a.Major != b.Major:
		if a.Major > b.Major {
			return 1
		}
		return -1
	case a.Minor != b.Minor:
		if a.Minor > b.Minor {
			return 1
		}
		return -1
	case a.Patch != b.Patch:
		if a.Patch > b.Patch {
			return 1
		}
		return -1
	default:
		return 0
	}
}

// ParseVersion parses a string into a Version struct.
func ParseVersion(s string) (Version, error) {
	if !IsSemVer(s) {
		return Version{}, ErrInvalidVersion
	}
	parts := strings.Split(s, ".")

	var version Version
	for idx, part := range parts {
		num, err := strconv.ParseUint(part, 0, 64)
		if err != nil {
			return Version{}, ErrInvalidVersion
		}
		switch idx {
		case 0:
			version.Major = num
		case 1:
			version.Minor = num
		case 2:
			version.Patch = num
		}
	}
	return version, nil
}

// IsSemVer reports whether s is a valid semantic version (e.g. 0.1.0).
// Only non-negative integers separated by dots are accepted.
func IsSemVer(s string) bool {
	rgxPattern := regexp.MustCompile(`^(?P<major>0|[1-9]\d*)\.(?P<minor>0|[1-9]\d*)\.(?P<patch>0|[1-9]\d*)$`)
	return rgxPattern.MatchString(s)
}
