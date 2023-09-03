package main

import (
	"flag"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/walkure/go-wxbeacon2"
	loggerFactory "github.com/walkure/homeprobe/pkg/logger"

	"kernel.org/pub/linux/libs/security/libcap/cap"
)

var promAddr = flag.String("listen", ":9821", "OpenMetrics Exporter Listeing Address")
var aboveSeaLevel = flag.Float64("above_sea_level", 0, "Height above sea level")
var wxBeacon2ID = flag.String("wxbeacon", "", "WxBeacon2 Device ID")
var logLevel = flag.String("loglevel", "INFO", "Log Level")

func main() {
	flag.Parse()
	initLogger(*logLevel)
	c := cap.GetProc()

	logger := loggerFactory.GetLogger("main")

	logger.Info("procinfo", slog.String("cap", c.String()))

	if *wxBeacon2ID == "" {
		logger.Error("argument `wxbeacon` is mandatory")
		return
	}

	logger.Info("arguments",
		slog.String("listen", *promAddr),
		slog.String("wxBeacon", *wxBeacon2ID),
		slog.Float64("aboveSeaLevel", *aboveSeaLevel),
	)

	metrics := initEnvData()

	dev := wxbeacon2.NewDevice(*wxBeacon2ID, wxDataCallback)
	err := dev.WaitForReceiveData()
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to open device:%v", err))
		return
	}

	http.HandleFunc("/metrics", func(w http.ResponseWriter, _ *http.Request) {
		metrics.Write(w)
	})

	err = http.ListenAndServe(*promAddr, nil)

	logger.Error("http handler terminated", slog.String("err", err.Error()))

}
