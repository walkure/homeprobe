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
	"github.com/walkure/go-wosensors"
	loggerFactory "github.com/walkure/homeprobe/pkg/logger"
	"github.com/walkure/homeprobe/pkg/metrics"
	"github.com/walkure/homeprobe/pkg/revision"

	"kernel.org/pub/linux/libs/security/libcap/cap"
)

var promAddr = flag.String("listen", ":9821", "OpenMetrics Exporter Listeing Address")
var logLevel = flag.String("loglevel", "INFO", "Log Level")
var woSensorTHOId = flag.String("tho", "", "WoSensorTHO Device ID")

// name of binary file populated at build-time
var binName = ""

func main() {
	flag.Usage = revision.Usage(binName)
	flag.Parse()

	loggerFactory.InitalizeLogger(*logLevel)
	wosensors.SetLogger(loggerFactory.GetLogger("wosensors"))
	c := cap.GetProc()

	logger := loggerFactory.GetLogger("main")

	logger.Info("procinfo", slog.String("cap", c.String()))

	logger.Info("arguments",
		slog.String("listen", *promAddr),
		slog.String("tho", *woSensorTHOId),
	)

	data := NewMetrics(15*time.Minute, metrics.Labels{"place": "outside"})
	tho := NewTHO(*woSensorTHOId, data)

	if tho == nil {
		logger.Error("No WoSensor activated. exit.")
		os.Exit(1)
	}

	// Active scanning
	d, err := gatt.NewDevice()

	if err != nil {
		panic(err)
	}

	d.Handle(gatt.PeripheralDiscovered(tho.Handler(nil)))

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

	// register handler to DefaultServeMux
	http.HandleFunc("/metrics", func(w http.ResponseWriter, _ *http.Request) {
		data.Write(w)
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
