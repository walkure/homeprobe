package main

import (
	"fmt"
	"io"
	"math"
	"time"

	"periph.io/x/conn/v3/physic"

	"periph.io/x/devices/v3/bmxx80"
	"periph.io/x/devices/v3/ccs811"

	z19 "github.com/eternal-flame-AD/mh-z19"

	"go.uber.org/multierr"
)

func measureMetrics(bme *bmxx80.Dev, ccs *ccs811.Dev, z19dev io.ReadWriter, start time.Time) error {

	warmingUp := time.Now().Before(start)

	if warmingUp {
		logPrintf("Warming Up until:%v\n", start)
	}

	var errors error

	multierr.AppendInto(&errors, measureMHZ19(z19dev, warmingUp))

	inTemp, inHumid, err := measureBMxx80(bme, warmingUp)

	if err != nil {
		multierr.Append(errors, err)
	} else {
		multierr.AppendInto(&errors, measureCCS811(ccs, inTemp, inHumid, warmingUp))
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
	logPrintf("%8s(%8s) %10s %9s ", temp, env.Temperature, env.Pressure, env.Humidity)

	inTemp = float64(temp.Celsius())
	inHumid = float64(env.Humidity) / float64(physic.PercentRH)

	fmt.Printf("Disconfort Index:%g\n", round(calcDisconfortIndex(inTemp, inHumid), 2))

	if !warmingUp {

		homeTemp.WithLabelValues("inside").Set(inTemp)
		homeHumid.WithLabelValues("inside").Set(round(inHumid, 2))

		hPa := float64(env.Pressure) / float64(physic.Pascal*100)
		homePressure.WithLabelValues("inside").Set(round(hPa, 2))

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

func measureMHZ19(z19dev io.ReadWriter, warmingUp bool) error {
	concentration, err := z19.TakeReading(z19dev)
	if err != nil {
		return fmt.Errorf("Z19: %w", err)
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
