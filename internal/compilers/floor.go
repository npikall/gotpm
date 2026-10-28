package compilers

import "fmt"

// Requirement is a compiler requirement and the package stating it.
type Requirement struct {
	// Version is the lowest compiler the package compiles with, or "" when it
	// states none.
	Version string
	// Of is the package ref of the declared dependency stating the
	// requirement, "" for the project's own.
	Of string
}

// String names the requirement and who states it, as in "the compiler
// requirement 0.14.0 of @gotpm/dep:1.0.0".
func (r Requirement) String() string {
	if r.Of == "" {
		return "the compiler requirement " + r.Version
	}
	return "the compiler requirement " + r.Version + " of " + r.Of
}

// Floor is the highest of requirements, the lowest compiler satisfying them
// all; a tie goes to the first stating it. No requirement is no floor.
func Floor(requirements []Requirement) (Requirement, error) {
	var floor Requirement
	for _, requirement := range requirements {
		if requirement.Version == "" {
			continue
		}
		if err := validVersion(requirement.Version); err != nil {
			if requirement.Of != "" {
				return Requirement{}, fmt.Errorf("compiler requirement of %s: %w", requirement.Of, err)
			}
			return Requirement{}, fmt.Errorf("compiler requirement: %w", err)
		}
		if floor.Version == "" || compareVersions(requirement.Version, floor.Version) > 0 {
			floor = requirement
		}
	}
	return floor, nil
}
