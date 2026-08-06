// Package semver parses, compares and increments the three-part versions typst
// packages are numbered with.
package semver

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

const (
	MAJOR string = "major"
	MINOR string = "minor"
	PATCH string = "patch"
)

var (
	ErrInvalidVersion   = errors.New("invalid semantic version")
	ErrInvalidIncrement = errors.New("invalid version incrementation, must be one of [major|minor|patch]")
)

type Version struct {
	Major int
	Minor int
	Patch int
}

func (v *Version) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

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

// Compare orders v against other, for sorting and for picking the latest of a
// set. Returns 1 if v > other, -1 if v < other and 0 if both are equal.
func (v *Version) Compare(other *Version) int {
	switch {
	case v.Major != other.Major:
		return sign(v.Major - other.Major)
	case v.Minor != other.Minor:
		return sign(v.Minor - other.Minor)
	case v.Patch != other.Patch:
		return sign(v.Patch - other.Patch)
	default:
		return 0
	}
}

func Parse(version string) (*Version, error) {
	parts := strings.Split(version, ".")
	allowedParts := 3
	if len(parts) != allowedParts {
		return &Version{}, fmt.Errorf("%w: expected 3 dot-separated numbers, got %d", ErrInvalidVersion, len(parts))
	}

	major, err := parseComponent(parts[0])
	if err != nil {
		return &Version{}, fmt.Errorf("version %q: major: %w", version, err)
	}
	minor, err := parseComponent(parts[1])
	if err != nil {
		return &Version{}, fmt.Errorf("version %q: minor: %w", version, err)
	}
	patch, err := parseComponent(parts[2])
	if err != nil {
		return &Version{}, fmt.Errorf("version %q: patch: %w", version, err)
	}

	return &Version{Major: major, Minor: minor, Patch: patch}, nil
}

func sign(diff int) int {
	if diff > 0 {
		return 1
	}
	return -1
}

func parseComponent(s string) (int, error) {
	if len(s) > 1 && s[0] == '0' {
		return 0, fmt.Errorf("%w: leading zero not allowed: %q", ErrInvalidVersion, s)
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("%w: not a valid integer: %q", ErrInvalidVersion, s)
	}
	if n < 0 {
		return 0, fmt.Errorf("%w: must not be negative: %q", ErrInvalidVersion, s)
	}
	return n, nil
}

func IsValidVersion(version string) bool {
	_, err := Parse(version)
	return err == nil
}
