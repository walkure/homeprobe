package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/walkure/go-lpsensors"
	"github.com/walkure/homeprobe/pkg/revision"
	"periph.io/x/conn/v3/i2c/i2creg"
	"periph.io/x/devices/v3/bmxx80"
	"periph.io/x/devices/v3/ccs811"
	"periph.io/x/host/v3"
)

var promAddr = flag.String("listen", ":9821", "OpenMetrics Exporter Listeing Address")
var tempOffset = flag.Float64("temp_offset", 0, "Temperature offset")
var aboveSeaLevel = flag.Float64("above_sea_level", 0, "Height above sea level")
var logLevel = flag.String("loglevel", "INFO", "Log Level")

const (
	ccs811_bus      = 0x5b
	bme280_bus      = 0x76
	sht3x_bus       = 0x45
	lps_bus         = 0x5c
	warming_seconds = 30
)

// name of binary file populated at build-time
var binName = ""

func main() {

	flag.Usage = revision.Usage(binName)
	flag.Parse()

	logger := initLogger(*logLevel)

	if _, err := host.Init(); err != nil {
		panic(fmt.Sprint("i2c initialize error: ", err))
	}

	bus, err := i2creg.Open("")
	if err != nil {
		panic(fmt.Sprint("i2cbus error: ", err))
	}
	defer bus.Close()

	// initialize devices

	// 1st BMxx80
	var bmx *bmxx80.Dev
	bmx, err = bmxx80.NewI2C(bus, bme280_bus, &bmxx80.DefaultOpts)
	if err != nil {
		logger.Warn("BMxx80 open error", slog.Any("err", err))
		bmx = nil
	} else {
		defer bmx.Halt()
		logger.Info("BMxx80 activated")
	}

	// 2nd CCS811
	var ccs *ccs811.Dev
	ccs, err = ccs811.New(bus, &ccs811.Opts{
		Addr:               ccs811_bus,
		MeasurementMode:    ccs811.MeasurementModeConstant250,
		InterruptWhenReady: false, UseThreshold: false})
	if err != nil {
		logger.Warn("CCS811 open error", slog.Any("err", err))
		ccs = nil
	} else {
		// Start CCS811
		if err = ccs.StartSensorApp(); err != nil {
			logger.Error("CCS811 start error", slog.Any("err", err))
			ccs = nil
		} else {
			logger.Info("CCS811 activated")
		}
	}

	// 3rd SHT3x
	var sht *SHT3x
	sht, err = NewSHT3x(sht3x_bus)
	if err != nil {
		logger.Warn("SHT3x open error", slog.Any("err", err))
		sht = nil
	} else {
		defer sht.Close()
		// Reset SHT3x
		if err = sht.Reset(); err != nil {
			logger.Error("SHT3x start error", slog.Any("err", err))
			sht = nil
		} else {
			logger.Info("SHT3x activated")
		}
	}

	// 4th LPS331AP
	var lps *lpsensors.Dev
	lps, err = lpsensors.NewI2C(bus, lps_bus, nil)
	if err != nil {
		logger.Warn("LPS331AP open error", slog.Any("err", err))
		lps = nil
	} else {
		logger.Info("LPS331AP activated")
	}

	if bmx == nil && sht == nil && ccs == nil && lps == nil {
		panic("no sensor detected.")
	}

	logger.Info("Temperature offset set:", slog.Float64("offset", *tempOffset))

	start := time.Now().Add(warming_seconds * time.Second)

	http.HandleFunc("/metrics", func(w http.ResponseWriter, _ *http.Request) {
		if time.Now().Before(start) {
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintln(w, "")
			logger.Info("Warming up..", slog.String("leftSecs", time.Until(start).String()))
			return
		}

		result, err := measure(bmx, ccs, sht, lps)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			logger.Error("measurement error", slog.Any("err", err))
			io.WriteString(w, fmt.Sprintf("Error:%s\n", err.Error()))
			return
		}

		result.Write(w)

	})

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serv := &http.Server{
		Addr: *promAddr,
	}

	go func() {
		logger.Info("server listening", slog.String("address", serv.Addr))

		if err := serv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("stop serving", slog.String("error", err.Error()))
		}
	}()
	<-ctx.Done()

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
