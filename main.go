package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"periph.io/x/conn/v3/i2c/i2creg"
	"periph.io/x/host/v3"

	"periph.io/x/devices/v3/bmxx80"
	"periph.io/x/devices/v3/ccs811"

	z19 "github.com/eternal-flame-AD/mh-z19"
	"github.com/tarm/serial"

	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/walkure/go-wxbeacon2"

	"kernel.org/pub/linux/libs/security/libcap/cap"
)

var promAddr = flag.String("listen", ":9821", "OpenMetrics Exporter Listeing Address")
var co2Addr = flag.String("mhz19", "/dev/ttyUSB0", "MH-Z19 UART Port")
var wxBeacon2ID = flag.String("wxbeacon", "", "WxBeacon2 Device ID")
var tempOffset = flag.Float64("temp_offset", 0, "BME280 Temperature offset")

const (
	ccs811_bus      = 0x5b
	bme280_bus      = 0x76
	warming_seconds = 30
)

var promReg = prometheus.NewRegistry()

var watchdog watchdogTimer

var homeTemp = promauto.NewGaugeVec(prometheus.GaugeOpts{
	Name: "temperature",
	Help: "Temperature",
}, []string{"place"})

var homeHumid = promauto.NewGaugeVec(prometheus.GaugeOpts{
	Name: "relative_humidity",
	Help: "Relative Humidity percent",
}, []string{"place"})

var homeAbsHumid = promauto.NewGaugeVec(prometheus.GaugeOpts{
	Name: "absolute_humidity",
	Help: "Absolute Humidity g/m^3",
}, []string{"place"})

var homePressure = promauto.NewGaugeVec(prometheus.GaugeOpts{
	Name: "pressure",
	Help: "Pressure",
}, []string{"place"})

var homeCO2 = promauto.NewGaugeVec(prometheus.GaugeOpts{
	Name: "co2",
	Help: "CO2 ppm",
}, []string{"place"})

var homeECO2 = promauto.NewGaugeVec(prometheus.GaugeOpts{
	Name: "eco2",
	Help: "eCO2 ppm",
}, []string{"place"})

var homeVOC = promauto.NewGaugeVec(prometheus.GaugeOpts{
	Name: "voc",
	Help: "VOC ppb",
}, []string{"place"})

var homeDisconfortIndex = promauto.NewGaugeVec(prometheus.GaugeOpts{
	Name: "disconfort_index",
	Help: "Disconfort Index",
}, []string{"place"})

var homeSoundNoise = promauto.NewGaugeVec(prometheus.GaugeOpts{
	Name: "sound_noise",
	Help: "Sound Noise db",
}, []string{"place"})

var homeHeatStroke = promauto.NewGaugeVec(prometheus.GaugeOpts{
	Name: "heat_stroke",
	Help: "WGBT",
}, []string{"place"})

var homeSensorVBat = promauto.NewGaugeVec(prometheus.GaugeOpts{
	Name: "sensor_vbat",
	Help: "Voltage of Sensor battery",
}, []string{"place"})

var homeAmbientLight = promauto.NewGaugeVec(prometheus.GaugeOpts{
	Name: "ambient_light",
	Help: "Ambient Light lx",
}, []string{"place"})

var homeUVIndex = promauto.NewGaugeVec(prometheus.GaugeOpts{
	Name: "uv_index",
	Help: "Index of UV",
}, []string{"place"})

func init() {
	promReg.MustRegister(homeTemp)
	promReg.MustRegister(homeHumid)
	promReg.MustRegister(homeAbsHumid)
	promReg.MustRegister(homePressure)
	promReg.MustRegister(homeCO2)
	promReg.MustRegister(homeECO2)
	promReg.MustRegister(homeVOC)

	promReg.MustRegister(homeDisconfortIndex)
	promReg.MustRegister(homeSoundNoise)
	promReg.MustRegister(homeHeatStroke)
	promReg.MustRegister(homeSensorVBat)

	promReg.MustRegister(homeAmbientLight)
	promReg.MustRegister(homeUVIndex)
}

func main() {

	c := cap.GetProc()
	log.Printf("this process has these capabilities:", c)

	// load arguments
	flag.Parse()

	// Load all the drivers:
	if _, err := host.Init(); err != nil {
		log.Fatal("initialize error: ", err)
	}

	if *wxBeacon2ID == "" {
		log.Fatal("WxBeacon2 ID not set")
	} else {
		log.Printf("WxBeacon2 ID:[%s]\n", *wxBeacon2ID)
	}

	if err := wxbeacon2.WaitForReceiveData(*wxBeacon2ID, receiveWxBeacon); err != nil {
		log.Fatal("WxBeacon2 error: ", err)
	}

	// Open a handle to the first available I²C bus:
	bus, err := i2creg.Open("")
	if err != nil {
		log.Fatal("i2cbus error: ", err)
	}
	defer bus.Close()

	// Open a handle to a bme280/bmp280 connected on the I²C bus using default
	// settings:
	dev, err := bmxx80.NewI2C(bus, bme280_bus, &bmxx80.DefaultOpts)
	if err != nil {
		log.Fatal("BMxx80 error: ", err)
	}
	defer dev.Halt()

	log.Printf("Temperature offset:%g\n", *tempOffset)

	// Open CCS811
	ccs, err := ccs811.New(bus, &ccs811.Opts{
		Addr:               ccs811_bus,
		MeasurementMode:    ccs811.MeasurementModeConstant250,
		InterruptWhenReady: false, UseThreshold: false})
	if err != nil {
		log.Fatal("CCS811 open error: ", err)
	}

	// Start CCS811
	if err = ccs.StartSensorApp(); err != nil {
		log.Fatal("CCS811 start error: ", err)
	}

	// Open MH-Z19
	log.Printf("MH-Z19 Device:[%s]\n", *co2Addr)
	connConfig := z19.CreateSerialConfig()
	connConfig.Name = *co2Addr
	port, err := serial.OpenPort(connConfig)
	if err != nil {
		log.Fatal("MH-Z19 open error: ", err)
	}
	defer port.Close()

	go func() {
		start := time.Now().Add(warming_seconds * time.Second)
		for {
			if err := measureMetrics(dev, ccs, port, start); err != nil {
				log.Printf("Error:%+v", err)
			}
			time.Sleep(time.Second * 15)
		}
	}()

	watchdog.Update()
	go func() {
		for {
			if watchdog.IsElapsed(time.Minute * 4) {
				log.Fatal("Watchdog expired!")
			}
			time.Sleep(time.Minute)
		}
	}()

	log.Printf("Listen [%s]\n", *promAddr)
	http.Handle("/metrics", promhttp.HandlerFor(promReg, promhttp.HandlerOpts{}))
	http.ListenAndServe(*promAddr, nil)
}

func logPrintf(format string, v ...interface{}) {
	now := time.Now().Format("2006-01-02 15:04:05")
	msg := fmt.Sprintf(format, v...)
	fmt.Printf("%s %s", now, msg)
}

func receiveWxBeacon(data interface{}) {
	switch v := data.(type) {
	case wxbeacon2.WxEPData:
		logPrintf("DataType:EP SeqNo:%d ", v.Sequence)
		fmt.Printf("Temp:%g ", v.Temp)
		fmt.Printf("Humid:%g ", v.Humid)
		fmt.Printf("AmbientLight:%d ", v.AmbientLight)
		fmt.Printf("UV Index:%g ", v.UVIndex)
		fmt.Printf("Pressure:%g ", v.Pressure)
		fmt.Printf("SoundNoise:%g ", v.SoundNoise)
		fmt.Printf("DisconfortIndex:%g(%g) ", v.DisconfortIndex, round(calcDisconfortIndex(v.Temp, v.Humid), 2))
		fmt.Printf("HeatStroke:%g ", v.HeatStroke)
		fmt.Printf("Battery:%gV\n", v.VBattery)

		homeTemp.WithLabelValues("outside").Set(v.Temp)
		homeHumid.WithLabelValues("outside").Set(v.Humid)
		homePressure.WithLabelValues("outside").Set(v.Pressure)

		homeDisconfortIndex.WithLabelValues("outside").Set(v.DisconfortIndex)
		homeSoundNoise.WithLabelValues("outside").Set(v.SoundNoise)
		homeHeatStroke.WithLabelValues("outside").Set(v.HeatStroke)
		homeAmbientLight.WithLabelValues("outside").Set(float64(v.AmbientLight))
		homeUVIndex.WithLabelValues("outside").Set(v.UVIndex)
		homeSensorVBat.WithLabelValues("outside").Set(v.VBattery)
		absHumid := calcAbsoluteHumidity(v.Temp, v.Humid)
		homeAbsHumid.WithLabelValues("outside").Set(round(absHumid, 2))

		watchdog.Update()
	}
}
