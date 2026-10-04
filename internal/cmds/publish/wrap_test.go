package publish //nolint: testpackage // wrap is unexported

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

var errGit = errors.New("git failed")

func TestWrap(t *testing.T) {
	t.Parallel()
	require.NoError(t, wrap(nil, "checking out %q", "foo"))

	err := wrap(errGit, "checking out %q", "foo")
	require.ErrorIs(t, err, errGit)
	require.EqualError(t, err, `checking out "foo": git failed`)
}
