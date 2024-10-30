package main

import (
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

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

	if bmx == nil && sht == nil && ccs == nil {
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

		result, err := measure(bmx, ccs, sht)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			logger.Error("measurement error", slog.Any("err", err))
			io.WriteString(w, fmt.Sprintf("Error:%s\n", err.Error()))
			return
		}

		result.Write(w)

	})

	logger.Error("Server stop", slog.Any("err", http.ListenAndServe(*promAddr, nil)))

}
