package devicemapper

import (
	"fmt"
	"strings"
	"github.com/Sirupsen/logrus"
)

// Working "notice" level into mappings as that is not a direct mapping for logrus log levels.
// definitions from lvm2 lib/log/log.h
const (
	LogLevelFatal  = 2 + iota // _LOG_FATAL
	LogLevelErr               // _LOG_ERR
	LogLevelWarn              // _LOG_WARN
	LogLevelNotice            // _LOG_NOTICE
	LogLevelInfo              // _LOG_INFO
	LogLevelDebug             // _LOG_DEBUG
)

var logFunc = []func(format string, args ...interface{}) {
	logrus.Infof,
	logrus.Infof,
	logrus.Fatalf,
	logrus.Errorf,
	logrus.Warnf,
	logrus.Infof,
	logrus.Infof,
	logrus.Debugf,
}

// ParseLevel takes a string level and returns the libdevicemapper log level constant.
func ParseLevel(lvl string) (int, error) {
	switch strings.ToLower(lvl) {
	case "fatal":
		return LogLevelFatal, nil
	case "err", "error":
		return LogLevelErr, nil
	case "warn", "warning":
		return LogLevelWarn, nil
	case "notice":
		return LogLevelNotice, nil
	case "info":
		return LogLevelInfo, nil
	case "debug":
		return LogLevelDebug, nil
	}

	return LogLevelInfo, fmt.Errorf("not a valid libdevicemapper log level: %q", lvl)
}

// Convert the Level to a string. E.g. LogLevelFatal becomes "fatal".
func FormatLevel(lvl int) string {
	switch lvl {
	case LogLevelDebug:
		return "debug"
	case LogLevelInfo:
		return "info"
	case LogLevelNotice:
		return "notice"
	case LogLevelWarn:
		return "warning"
	case LogLevelErr:
		return "error"
	case LogLevelFatal:
		return "fatal"
	}

	return "info"
}

func Logf(level int, file string, line int, dmError int, message string) {
	f := logrus.Debugf
	if level < LogLevelDebug {
		f = logFunc[level]
	}

	f("libdevmapper(%s): %s:%d (%d) %s", FormatLevel(level), file, line, dmError, message)
}
