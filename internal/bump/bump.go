package bump

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// A Semantic Version struct, where
// only positive integers are allowed
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

// Bump the Version by the given increment (major, minor, patch)
// Returns an ErrInvalidIncrement if the wrong increment is used.
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

// Create a new Version struct with all Fields set to Zero
func NewVersion() Version {
	return Version{Major: 0, Minor: 0, Patch: 0}
}

// Parse a string into a Version Struct
func ParseVersion(s string) (Version, error) {
	isSemVer := IsSemVer(s)
	if !isSemVer {
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

// Check if a given string is a valid semantic version (e.g. 0.1.0)
//   - No Letters, just positive Numbers
//   - 3 Components (Numbers) separated with '.'
func IsSemVer(s string) bool {
	rgxPattern := regexp.MustCompile("^(0|[1-9]d*).(0|[1-9]d*).(0|[1-9]d*)$")
	return rgxPattern.MatchString(s)
}
