package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"time"

	"periph.io/x/conn/v3/i2c/i2creg"
	"periph.io/x/devices/v3/bmxx80"
	"periph.io/x/devices/v3/ccs811"
	"periph.io/x/host/v3"
)

var promAddr = flag.String("listen", ":9821", "OpenMetrics Exporter Listeing Address")
var tempOffset = flag.Float64("temp_offset", 0, "Temperature offset")
var aboveSeaLevel = flag.Float64("above_sea_level", 0, "Height above sea level")

const (
	ccs811_bus      = 0x5b
	bme280_bus      = 0x76
	sht3x_bus       = 0x45
	warming_seconds = 30
)

func main() {
	flag.Parse()

	if _, err := host.Init(); err != nil {
		log.Fatal("i2c initialize error: ", err)
	}

	bus, err := i2creg.Open("")
	if err != nil {
		log.Fatal("i2cbus error: ", err)
	}
	defer bus.Close()

	// initialize devices

	// 1st BMxx80
	var bmx *bmxx80.Dev
	bmx, err = bmxx80.NewI2C(bus, bme280_bus, &bmxx80.DefaultOpts)
	if err != nil {
		log.Printf("BMxx80 error: ", err)
		bmx = nil
	} else {
		defer bmx.Halt()
		log.Printf("BMxx80 activated\n")
	}

	// 2nd CCS811
	var ccs *ccs811.Dev
	ccs, err = ccs811.New(bus, &ccs811.Opts{
		Addr:               ccs811_bus,
		MeasurementMode:    ccs811.MeasurementModeConstant250,
		InterruptWhenReady: false, UseThreshold: false})
	if err != nil {
		log.Printf("CCS811 open error: ", err)
		ccs = nil
	} else {
		// Start CCS811
		if err = ccs.StartSensorApp(); err != nil {
			log.Printf("CCS811 start error: ", err)
			ccs = nil
		}
		log.Printf("CCS811 activated\n")
	}

	// 3rd SHT3x
	var sht *SHT3x
	sht, err = NewSHT3x(sht3x_bus)
	if err != nil {
		log.Printf("SHT3x open error: ", err)
		sht = nil
	} else {
		defer sht.Close()
		// Reset SHT3x
		if err = sht.Reset(); err != nil {
			log.Printf("SHT3x start error: ", err)
			sht = nil
		}
		log.Printf("SHT3x activated\n")
	}
	if bmx == nil && sht == nil {
		log.Fatal("no sensor detected.")
	}

	log.Printf("Temperature offset:%g\n", *tempOffset)

	start := time.Now().Add(warming_seconds * time.Second)

	http.HandleFunc("/metrics", func(w http.ResponseWriter, _ *http.Request) {
		if time.Now().Before(start) {
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintln(w, "")
			log.Printf("Warming up... %d seconds left\n", start.Sub(time.Now())/time.Second)
			return
		}

		result, err := measure(bmx, ccs, sht)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Printf("Error:%s\n", err.Error())
			io.WriteString(w, fmt.Sprintf("Error:%s\n", err.Error()))
			return
		}

		if !math.IsNaN(result.inTemp) {
			fmt.Fprintln(w, "# HELP temperature Temperature")
			fmt.Fprintln(w, "# TYPE temperature gauge")
			fmt.Fprintf(w, "temperature{place=\"inside\"} %s\n", roundFormat(result.inTemp, 2))
		}

		if !math.IsNaN(result.inHumid) {
			fmt.Fprintln(w, "# HELP relative_humidity Relative Humidity percent")
			fmt.Fprintln(w, "# TYPE relative_humidity gauge")
			fmt.Fprintf(w, "relative_humidity{place=\"inside\"} %s\n", roundFormat(result.inHumid, 2))
		}

		if !math.IsNaN(result.inHumidAbs) {
			fmt.Fprintln(w, "# HELP absolute_humidity Absolute Humidity g/m^3")
			fmt.Fprintln(w, "# TYPE absolute_humidity gauge")
			fmt.Fprintf(w, "absolute_humidity{place=\"inside\"} %s\n", roundFormat(result.inHumidAbs, 2))
		}

		if !math.IsNaN(result.disconfortIndex) {
			fmt.Fprintln(w, "# HELP disconfort_index Disconfort Index")
			fmt.Fprintln(w, "# TYPE disconfort_index gauge")
			fmt.Fprintf(w, "disconfort_index{place=\"inside\"} %s\n", roundFormat(result.disconfortIndex, 2))
		}

		if !math.IsNaN(result.hPaMSL) {
			fmt.Fprintln(w, "# HELP pressure Pressure hPa")
			fmt.Fprintln(w, "# TYPE pressure gauge")
			fmt.Fprintf(w, "pressure{place=\"inside\"} %s\n", roundFormat(result.hPaMSL, 2))
		}

		if !math.IsNaN(result.eCO2) {
			fmt.Fprintln(w, "# HELP eco2 eCO2 ppm")
			fmt.Fprintln(w, "# TYPE eco2 gauge")
			fmt.Fprintf(w, "eCO2{place=\"inside\"} %f\n", result.eCO2)
		}

		if !math.IsNaN(result.voc) {
			fmt.Fprintln(w, "# HELP voc VOC ppb")
			fmt.Fprintln(w, "# TYPE voc gauge")
			fmt.Fprintf(w, "voc{place=\"inside\"} %f\n", result.voc)
		}

	})

	log.Fatal(http.ListenAndServe(*promAddr, nil))
}
