package locate

import (
	"charm.land/log/v2"
	"github.com/npikall/gotpm/internal/config"
	"github.com/npikall/gotpm/internal/deps"
	"github.com/npikall/gotpm/internal/index"
	"github.com/npikall/gotpm/internal/lockfile"
	"github.com/npikall/gotpm/internal/paths"
	"github.com/npikall/gotpm/internal/remote"
)

// Entry is a single path gotpm reads or writes.
//
// Resolving a path can fail on its own — an unset %APPDATA% takes the data
// directory with it but leaves the config directory intact — so a failure is
// carried in Err rather than aborting the whole inventory.
type Entry struct {
	// Key addresses the entry on the command line.
	Key string
	// Path is where the entry points, empty when Err is set.
	Path string
	// Note explains the path, e.g. which environment variable set it.
	Note string
	// Err is why the path could not be resolved.
	Err error
}

// Group is a set of entries printed under a common heading.
type Group struct {
	Name    string
	Entries []Entry
}

// Keys returns every key the command accepts, in the order they are printed.
// It is fixed rather than derived from the current directory, so shell
// completion offers the same keys inside and outside a project.
func Keys() []string {
	return []string{
		"packages",
		"data-dir", "config-dir", "config", "index", "remotes",
		"root", "manifest", "lock",
	}
}

// entries returns every path gotpm knows about. The project group is only
// present when the working directory belongs to a typst project.
func entries(log *log.Logger) []Group {
	groups := make([]Group, 0, 3) //nolint: mnd
	groups = append(groups,
		Group{Name: "Typst", Entries: []Entry{packagesEntry()}},
		Group{Name: "gotpm", Entries: gotpmEntries()},
	)
	project, err := projectEntries()
	if err != nil {
		log.Debug("no project paths", "reason", err)
		return groups
	}
	return append(groups, Group{Name: "project", Entries: project})
}

func packagesEntry() Entry {
	dir, origin, err := paths.PackagesDir()
	entry := Entry{Key: "packages", Path: dir, Err: err}
	if env := origin.EnvVar(); env != "" {
		entry.Note = "via $" + env
	}
	return entry
}

func gotpmEntries() []Entry {
	dataDir, dataErr := paths.GotpmDataDir()
	configDir, configErr := paths.GotpmConfigDir()
	configFile, configFileErr := config.Path()
	indexCache, indexErr := index.CachePath()
	remotes, remotesErr := remote.CacheDir()

	return []Entry{
		{Key: "data-dir", Path: dataDir, Err: dataErr},
		{Key: "config-dir", Path: configDir, Err: configErr},
		{Key: "config", Path: configFile, Err: configFileErr},
		{Key: "index", Path: indexCache, Err: indexErr},
		{Key: "remotes", Path: remotes, Err: remotesErr},
	}
}

// projectEntries returns the project paths. A malformed manifest errors the
// same as no project at all: either way there is nothing trustworthy to point
// at. The dump swallows the error and drops the group; asking for a project
// key by name reports it.
func projectEntries() ([]Entry, error) {
	project, err := deps.OpenProject()
	if err != nil {
		return nil, err
	}
	return []Entry{
		{Key: "root", Path: project.Dir},
		{Key: "manifest", Path: project.File},
		{Key: "lock", Path: lockfile.Path(project.Dir)},
	}, nil
}
