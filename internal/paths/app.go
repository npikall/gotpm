package paths

// GotpmDataDir returns the platform specific path to gotpm's data directory
func GotpmDataDir() (string, error) {
	return AppDataDir("gotpm")
}

// GotpmConfigDir returns the platform specific path to gotpm's config directory
func GotpmConfigDir() (string, error) {
	return AppConfigDir("gotpm")
}
