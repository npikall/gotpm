//go:build !unix

package run

import (
	"errors"
	"runtime"
)

var errUnsupported = errors.New("gotpm run is not supported on " + runtime.GOOS + " yet")

func launch(Launch) error {
	return errUnsupported
}
