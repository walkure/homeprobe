package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	z19 "github.com/eternal-flame-AD/mh-z19"
	"github.com/tarm/serial"
	loggerFactory "github.com/walkure/homeprobe/pkg/logger"
	"github.com/walkure/homeprobe/pkg/revision"
)

var mhz19Addr = flag.String("mhz19", "", "MH-Z19 UART Port")
var promAddr = flag.String("listen", ":9821", "OpenMetrics Exporter Listeing Address")
var logLevel = flag.String("loglevel", "INFO", "Log Level")

const warmingSeconds = 30

// name of binary file populated at build-time
var binName = ""

func main() {

	flag.Usage = revision.Usage(binName)
	flag.Parse()

	logger := loggerFactory.InitalizeLogger(*logLevel)

	log.Printf("MH-Z19 device:[%s]\n", *mhz19Addr)
	if *mhz19Addr == "" {
		panic("MH-Z19 device not specified")
	}

	start := time.Now().Add(warmingSeconds * time.Second)

	http.HandleFunc("/metrics", func(w http.ResponseWriter, _ *http.Request) {

		if time.Now().Before(start) {
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintln(w, "# HELP co2 CO2 ppm")
			fmt.Fprintln(w, "# TYPE co2 gauge")
			logger.Info("Warming up..", slog.String("leftSecs", time.Until(start).String()))
			return
		}

		concentration, err := measureMHZ19()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			logger.Error("measurement error", slog.Any("err", err))
			io.WriteString(w, fmt.Sprintf("Error:%s\n", err.Error()))
			return
		}
		fmt.Fprintln(w, "# HELP co2 CO2 ppm")
		fmt.Fprintln(w, "# TYPE co2 gauge")
		fmt.Fprintf(w, "co2{place=\"inside\"} %d\n", concentration)
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

func measureMHZ19() (uint16, error) {

	connConfig := z19.CreateSerialConfig()
	connConfig.Name = *mhz19Addr
	connConfig.ReadTimeout = time.Second * 5

	mhz, err := serial.OpenPort(connConfig)
	if err != nil {
		return 0, fmt.Errorf("MH-Z19B[%s] cannot open:%w", *mhz19Addr, err)
	}

	defer mhz.Close()

	concentration, err := z19.TakeReading(mhz)

	if err != nil {
		return 0, fmt.Errorf("MH-Z19B[%s] read error:%w", *mhz19Addr, err)
	}

	return concentration, nil
}
