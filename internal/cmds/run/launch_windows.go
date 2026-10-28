//go:build windows

package run

// launch runs the compiler as a child process: windows cannot replace one.
func launch(l Launch) error {
	return launchChild(l)
}
