package main

import (
	"log/slog"
	"os"

	"github.com/walkure/go-wxbeacon2"
)

var logger = slog.Default()

func newLogger(level string) {
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

	wxbeacon2.SetLogger(logger)
}
