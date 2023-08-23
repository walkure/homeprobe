package main

import (
	"fmt"
	"github.com/tarm/serial"
	"math"
	"time"
	"log"

	"periph.io/x/conn/v3/physic"

	"periph.io/x/devices/v3/bmxx80"
	"periph.io/x/devices/v3/ccs811"

	sht3x "github.com/d2r2/go-sht3x"
	z19 "github.com/eternal-flame-AD/mh-z19"

	"go.uber.org/multierr"
)

func measureMetrics(bme *bmxx80.Dev, ccs *ccs811.Dev, sht *SHT3x, start time.Time) error {

	warmingUp := time.Now().Before(start)

	if warmingUp {
		logPrintf("Warming Up until:%v\n", start)
	}
	fmt.Printf("BME %p,CCS %p,SHT %p\n",bme,ccs,sht)

	var errors error

	if *co2Addr != "" {
		logPrintf("Begin MHZ19\n")
		multierr.AppendInto(&errors, measureMHZ19(warmingUp))
		logPrintf("End MHZ19\n")
	}else{
		logPrintf("No MHZ19\n")
	}

	var inTemp, inHumid float64
	var err error

	if bme != nil {
		logPrintf("Begin BMxx80\n")
		inTemp, inHumid, err = measureBMxx80(bme, warmingUp)
		fmt.Printf("BME2xx Temp:%v Humid:%v\n",inTemp,inHumid)
		if err != nil {
			multierr.AppendInto(&errors, err)
		}
	}

	if sht != nil {
		logPrintf("Begin SHT3x\n")
		inTemp, inHumid, err = measureSHT3x(sht, warmingUp)
		fmt.Printf("SHT3x Temp:%v Humid:%v\n",inTemp,inHumid)
		if err != nil {
			multierr.AppendInto(&errors, err)
		}
	}

	if ccs != nil && (sht != nil || bme != nil) {
		logPrintf("BeginCS811\n")
		multierr.AppendInto(&errors, measureCCS811(ccs, inTemp, inHumid, warmingUp))
		logPrintf("END CCS811\n")
	}

	return errors

}

func measureBMxx80(bme *bmxx80.Dev, warmingUp bool) (inTemp, inHumid float64, err error) {
	var env physic.Env
	if err = bme.Sense(&env); err != nil {
		err = fmt.Errorf("BME: %w", err)
		return
	}
	temp := env.Temperature + physic.Temperature(*tempOffset)*physic.Celsius
	logPrintf("BME2xx %8s(%8s) %10s %9s ", temp, env.Temperature, env.Pressure, env.Humidity)

	inTemp = float64(temp.Celsius())
	inHumid = float64(env.Humidity) / float64(physic.PercentRH)

	fmt.Printf("Disconfort Index:%g\n", round(calcDisconfortIndex(inTemp, inHumid), 2))

	if !warmingUp {

		homeTemp.WithLabelValues("inside").Set(inTemp)
		homeHumid.WithLabelValues("inside").Set(round(inHumid, 2))

		hPaMSL := calcMeanHeightAirPressure(float64(env.Pressure)/float64(physic.Pascal*100), inTemp, *aboveSeaLevel)
		homePressure.WithLabelValues("inside").Set(round(hPaMSL, 2))

		absHumid := calcAbsoluteHumidity(inTemp, inHumid)
		homeAbsHumid.WithLabelValues("inside").Set(round(absHumid, 2))

		disconfortIndex := calcDisconfortIndex(inTemp, inHumid)
		homeDisconfortIndex.WithLabelValues("inside").Set(round(disconfortIndex, 2))
	}

	return
}

func measureSHT3x(sht *SHT3x, warmingUp bool) (inTemp, inHumid float64, err error) {
	inTemp, inHumid, err = sht.ReadTemperatureAndRelativeHumidity(sht3x.RepeatabilityMedium)
	if err != nil {
		err = fmt.Errorf("SHT3x: %w", err)
		return
	}
	logPrintf("SHT3x %v*C, %v%%\n", inTemp, inHumid)

	fmt.Printf("Disconfort Index:%g\n", round(calcDisconfortIndex(inTemp, inHumid), 2))

	if !warmingUp {

		homeTemp.WithLabelValues("inside").Set(inTemp)
		homeHumid.WithLabelValues("inside").Set(round(inHumid, 2))

		absHumid := calcAbsoluteHumidity(inTemp, inHumid)
		homeAbsHumid.WithLabelValues("inside").Set(round(absHumid, 2))

		disconfortIndex := calcDisconfortIndex(inTemp, inHumid)
		homeDisconfortIndex.WithLabelValues("inside").Set(round(disconfortIndex, 2))
	}

	return
}

func measureCCS811(ccs *ccs811.Dev, inTemp float64, inHumid float64, warmingUp bool) error {

	if err := ccs.SetEnvironmentData(float32(inTemp), float32(inHumid)); err != nil {
		return fmt.Errorf("CCS init: %w", err)
	}

	var air ccs811.SensorValues
	if err := ccs.SensePartial(ccs811.ReadCO2VOCStatus, &air); err != nil {
		return fmt.Errorf("CCS: %w", err)
	}

	logPrintf("eCO2:%dppm VOC:%dppb\n", air.ECO2, air.VOC)

	if !warmingUp {
		homeECO2.WithLabelValues("inside").Set(float64(air.ECO2))
		homeVOC.WithLabelValues("inside").Set(float64(air.VOC))
	}

	return nil
}

func measureMHZ19(warmingUp bool) error {
	connConfig := z19.CreateSerialConfig()
	connConfig.Name = *co2Addr
	connConfig.ReadTimeout = time.Second * 5

	mhz, err := serial.OpenPort(connConfig)
	if err != nil {
		log.Printf("MH-Z19 open error: ", err)
		return fmt.Errorf("MH-Z19 open error:%w",err)
	}
	defer mhz.Close()
	logPrintf("MH-Z19 activated\n")

	concentration, err := z19.TakeReading(mhz)
	logPrintf("EndMHZ19Read\n")
	if err != nil {
		log.Printf("MH-Z19 read error: ", err)
		return fmt.Errorf("MH-Z19 read error: %w", err)
	}

	logPrintf("co2=%dppm\n", concentration)

	if !warmingUp {
		homeCO2.WithLabelValues("inside").Set(float64(concentration))
	}

	return nil
}

func calcAbsoluteHumidity(temp, relativeHumid float64) float64 {
	vaporPressureSat := 6.1078 * math.Pow(10, 7.5*temp/(temp+237.7))
	vaporAmountSat := 217 * vaporPressureSat / (temp + 273.15)

	return vaporAmountSat * relativeHumid / 100
}

func calcDisconfortIndex(temp, relativeHumid float64) float64 {
	return 0.81*temp + 0.01*relativeHumid*(0.99*temp-14.3) + 46.3
}

func round(value float64, places int) float64 {
	shift := math.Pow10(places)
	return math.Round(value*shift) / shift
}

func calcMeanHeightAirPressure(pressure, temp, height float64) float64 {
	kelvin := temp + 273.15
	return pressure * math.Pow(kelvin/(kelvin+0.0065*height), -5.257)
}
