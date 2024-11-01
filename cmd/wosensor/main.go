package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/walkure/gatt"
	"github.com/walkure/go-wosensors"
	loggerFactory "github.com/walkure/homeprobe/pkg/logger"
	"github.com/walkure/homeprobe/pkg/metrics"
	"github.com/walkure/homeprobe/pkg/revision"
	"github.com/walkure/homeprobe/pkg/weather"

	"kernel.org/pub/linux/libs/security/libcap/cap"
)

var promAddr = flag.String("listen", ":9821", "OpenMetrics Exporter Listeing Address")
var woSensorTHOId = flag.String("tho", "", "WoSensorTHO Device ID")
var logLevel = flag.String("loglevel", "INFO", "Log Level")

// name of binary file populated at build-time
var binName = ""

func main() {
	flag.Usage = revision.Usage(binName)
	flag.Parse()

	loggerFactory.InitalizeLogger(*logLevel)
	wosensors.SetLogger(loggerFactory.GetLogger("wosensor_tho"))
	c := cap.GetProc()

	logger := loggerFactory.GetLogger("main")

	logger.Info("procinfo", slog.String("cap", c.String()))

	if *woSensorTHOId == "" {
		logger.Error("argument `tho` is mandatory")
		return
	}

	logger.Info("arguments",
		slog.String("listen", *promAddr),
		slog.String("tho", *woSensorTHOId),
	)

	data := metrics.MetricSet{}

	temp := metrics.NewGauge("temperature", "Temperature")
	relHumid := metrics.NewGauge("relative_humidity", "Relative Humidity percent")
	absHumid := metrics.NewGauge("absolute_humidity", "Absolute Humidity g/m^3")
	disconfortIndex := metrics.NewGauge("disconfort_index", "Disconfort Index")
	vBattery := metrics.NewGauge("sensor_vbat", "Voltage of Sensor battery")
	data.Add(temp, relHumid, absHumid, disconfortIndex, vBattery)

	// Active scanning
	d, err := gatt.NewDevice()

	if err != nil {
		panic(err)
	}

	var mu sync.Mutex
	seqno := uint8(0)

	labels := metrics.Labels{"place": "outside"}

	d.Handle(gatt.PeripheralDiscovered(wosensors.HandleWoSensorTHO(*woSensorTHOId, false, func(d wosensors.THOData) {
		mu.Lock()
		defer mu.Unlock()
		if seqno == d.SequenceNumber {
			logger.Debug("sequence not changed", slog.Uint64("seq", uint64(d.SequenceNumber)))
			return
		}

		logger.Debug("new data", "d", d, "seq", d.SequenceNumber)

		seqno = d.SequenceNumber

		expireAt := time.Now().Add(15 * time.Minute)
		temp.SetWithTimeout(
			labels,
			metrics.RoundFloat64{
				Value:     float64(d.Temperature),
				Precision: 2,
			},
			expireAt,
		)
		relHumid.SetWithTimeout(
			labels,
			metrics.RoundFloat64{
				Value:     float64(d.Humidity),
				Precision: 0,
			},
			expireAt,
		)
		absHumid.SetWithTimeout(
			labels,
			metrics.RoundFloat64{
				Value:     weather.AbsoluteHumidity(float64(d.Temperature), float64(d.Humidity)),
				Precision: 2,
			},
			expireAt,
		)
		disconfortIndex.SetWithTimeout(
			labels,
			metrics.RoundFloat64{
				Value:     weather.DisconfortIndex(float64(d.Temperature), float64(d.Humidity)),
				Precision: 2,
			},
			expireAt,
		)
		if d.BatteryPercent <= 100 {
			vBattery.SetWithTimeout(
				labels,
				metrics.RoundFloat64{
					Value:     float64(d.BatteryPercent) / 100.0 * 3,
					Precision: 3,
				},
				expireAt,
			)
		}

		logger.Info("updated", "d", data, "seq", d.SequenceNumber)

	}, nil)))

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
