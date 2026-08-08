package logger_test

import (
	"testing"

	"charm.land/log/v2"
	"github.com/npikall/gotpm/internal/logger"
	"github.com/stretchr/testify/assert"
)

func TestSetup(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		verbose   int
		wantLevel log.Level
	}{
		{"no flags → WarnLevel", 0, log.WarnLevel},
		{"one flag → InfoLevel", 1, log.InfoLevel},
		{"two flags → DebugLevel", 2, log.DebugLevel},
		{"three flags → DebugLevel", 3, log.DebugLevel},
		{"negative → WarnLevel", -1, log.WarnLevel},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.wantLevel, logger.Setup(tt.verbose).GetLevel())
		})
	}
}
