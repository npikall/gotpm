package index

import "testing"

// UseIndexURL points Fetch at u for the rest of the test.
func UseIndexURL(t *testing.T, u string) {
	t.Helper()
	previous := indexURL
	indexURL = u
	t.Cleanup(func() { indexURL = previous })
}

// UsePackageEndpoint points LatestOnGitHub at u for the rest of the test.
func UsePackageEndpoint(t *testing.T, u string) {
	t.Helper()
	previous := packageEndpoint
	packageEndpoint = u
	t.Cleanup(func() { packageEndpoint = previous })
}
