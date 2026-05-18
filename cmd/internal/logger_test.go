package internal

import (
	"strconv"
	"testing"

	"charm.land/log/v2"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newCmdWithVerbose(t *testing.T, count int) *cobra.Command {
	t.Helper()
	cmd := &cobra.Command{}
	cmd.Flags().CountP("verbose", "v", "verbose output")
	if count > 0 {
		require.NoError(t, cmd.Flags().Set("verbose", strconv.Itoa(count)))
	}
	return cmd
}

func TestSetupLogger(t *testing.T) {
	tests := []struct {
		name      string
		verbose   int
		wantLevel log.Level
	}{
		{"no flags → WarnLevel", 0, log.WarnLevel},
		{"one flag → InfoLevel", 1, log.InfoLevel},
		{"two flags → DebugLevel", 2, log.DebugLevel},
		{"three flags → DebugLevel", 3, log.DebugLevel},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := newCmdWithVerbose(t, tt.verbose)
			logger := SetupLogger(cmd)
			assert.Equal(t, tt.wantLevel, logger.GetLevel())
		})
	}
}

func TestSetupLogger_NoVerboseFlag(t *testing.T) {
	cmd := &cobra.Command{}
	logger := SetupLogger(cmd)
	assert.Equal(t, log.WarnLevel, logger.GetLevel())
}

func TestMust_ReturnsValue(t *testing.T) {
	got := Must("hello", nil)
	assert.Equal(t, "hello", got)

	n := Must(42, nil)
	assert.Equal(t, 42, n)
}
