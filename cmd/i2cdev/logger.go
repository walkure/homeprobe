package main

import (
	"log/slog"
	"strings"

	logger "github.com/d2r2/go-logger"
	loggerFactory "github.com/walkure/homeprobe/pkg/logger"
)

func initLogger(level string) *slog.Logger {
	l := loggerFactory.InitalizeLogger(level)

	lv := logger.InfoLevel

	lvStr := strings.ToUpper(level)

	if strings.HasPrefix(lvStr, "DEBUG") {
		lv = logger.DebugLevel
	} else if strings.HasPrefix(lvStr, "INFO") {
		lv = logger.InfoLevel
	} else if strings.HasPrefix(lvStr, "WARN") {
		lv = logger.WarnLevel
	} else if strings.HasPrefix(lvStr, "ERROR") {
		lv = logger.ErrorLevel
	}

	logger.ChangePackageLogLevel("i2c", lv)
	logger.ChangePackageLogLevel("sht3x", lv)

	return l
}
