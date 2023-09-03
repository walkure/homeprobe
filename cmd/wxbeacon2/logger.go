package main

import (
	"github.com/walkure/go-wxbeacon2"
	loggerFactory "github.com/walkure/homeprobe/pkg/logger"
)

func initLogger(level string) {
	loggerFactory.InitalizeLogger(level)
	wxbeacon2.SetLogger(loggerFactory.GetLogger("wxbeacon2"))
}
