package update

import "charm.land/log/v2"

type logEvent struct {
	level   string
	msg     string
	keyvals []any
}

func (e logEvent) emit(log *log.Logger) {
	switch e.level {
	case "debug":
		log.Debug(e.msg, e.keyvals...)
	case "info":
		log.Info(e.msg, e.keyvals...)
	case "error":
		log.Error(e.msg, e.keyvals...)
	}
}
