package config //nolint: testpackage // assign is reached only through string fields today

import (
	"reflect"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAssign pins the value types a config key can hold, ahead of the first
// non-string key.
func TestAssign(t *testing.T) {
	t.Parallel()
	var target struct {
		S string
		B bool
		I int64
		F float64
	}
	field := func(name string) reflect.Value { return reflect.ValueOf(&target).Elem().FieldByName(name) }

	require.NoError(t, assign(field("S"), "text"))
	require.NoError(t, assign(field("B"), "true"))
	require.NoError(t, assign(field("I"), "42"))
	assert.Equal(t, "text", target.S)
	assert.True(t, target.B)
	assert.Equal(t, int64(42), target.I)

	require.ErrorIs(t, assign(field("B"), "maybe"), strconv.ErrSyntax)
	require.ErrorIs(t, assign(field("I"), "many"), strconv.ErrSyntax)
	require.ErrorIs(t, assign(field("F"), "1.5"), ErrUnsupportedField)
}
