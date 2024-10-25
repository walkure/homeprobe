package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/walkure/gatt"
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

	// Passive scanning
	d, err := gatt.NewDevice(gatt.LnxSetScanMode(false))

	if err != nil {
		panic(err)
	}

	d.Handle(gatt.PeripheralDiscovered(
		wxbeacon2.HandleWxBeacon2(*wxBeacon2ID, wxDataCallback, nil)))

	d.Init(func(d gatt.Device, s gatt.State) {
		switch s {
		case gatt.StatePoweredOn:
			// allow duplicate
			d.Scan([]gatt.UUID{}, true)
			return
		default:
			d.StopScanning()
		}
	})

	serv := &http.Server{
		Addr: *promAddr,
	}

	http.HandleFunc("/metrics", func(w http.ResponseWriter, _ *http.Request) {
		metrics.Write(w)
	})

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		logger.Info("server listening", slog.String("address", serv.Addr))

		if err := serv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("stop serving", slog.String("error", err.Error()))
		}
	}()
	<-ctx.Done()

	d.StopScanning()
	if err := d.Stop(); err != nil {
		logger.Error("hci stop", slog.String("error", err.Error()))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	logger.Warn("shutting down server")

	if err := serv.Shutdown(ctx); err != nil {
		logger.Error("server shutdown", slog.String("error", err.Error()))
		if err := serv.Close(); err != nil {
			logger.Error("server close", slog.String("error", err.Error()))
		}
	}

}
