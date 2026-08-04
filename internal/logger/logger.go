package logger

import (
	"os"

	"charm.land/log/v2"
)

func Setup(level int) *log.Logger {
	logger := log.New(os.Stderr)
	logger.SetReportTimestamp(false)
	switch {
	case level >= 2: //nolint: mnd
		logger.SetLevel(log.DebugLevel)
	case level == 1:
		logger.SetLevel(log.InfoLevel)
	default:
		logger.SetLevel(log.WarnLevel)
	}
	return logger
}
