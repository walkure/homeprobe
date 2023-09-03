package logger

import (
	"log/slog"
	"os"
)

var logger = slog.Default()

func InitalizeLogger(level string) *slog.Logger {
	var lv slog.Level

	if level != "" {
		err := lv.UnmarshalText([]byte(level))
		if err != nil {
			lv = slog.LevelInfo
		}
	} else {
		lv = slog.LevelInfo
	}

	logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: lv,
	}))

	return logger
}

func GetLogger(category string) *slog.Logger {
	return logger.With(slog.String("category", category))
}
