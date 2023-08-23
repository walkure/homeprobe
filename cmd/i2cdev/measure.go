package main

import (
	"fmt"
	//"log"
	"math"

	"periph.io/x/conn/v3/physic"
	"periph.io/x/devices/v3/bmxx80"
	"periph.io/x/devices/v3/ccs811"

	sht3x "github.com/d2r2/go-sht3x"
)

func measure(bme *bmxx80.Dev, ccs *ccs811.Dev, sht *SHT3x) (float64, float64, float64, float64, float64, float64, float64, error) {

	var inTemp, inHumid, hPaMSL, eCO2, voc float64
	var err error

	if bme != nil {
		inTemp, inHumid, hPaMSL, err = measureBMxx80(bme)
		if err != nil {
			return 0, 0, 0, 0, 0, 0, 0, err
		}
	} else {
		hPaMSL = math.NaN()
	}

	if sht != nil {
		inTemp, inHumid, err = measureSHT3x(sht)
		if err != nil {
			return 0, 0, 0, 0, 0, 0, 0, err
		}
	}

	if ccs != nil {
		eCO2, voc, err = measureCCS811(inTemp, inHumid, ccs)
		if err != nil {
			return 0, 0, 0, 0, 0, 0, 0, err
		}
	} else {
		eCO2 = math.NaN()
		voc = math.NaN()
	}

	disconfortIndex := calcDisconfortIndex(inTemp, inHumid)
	inHumidAbs := calcAbsoluteHumidity(inTemp, inHumid)

	return inTemp, inHumid, inHumidAbs, disconfortIndex, hPaMSL, eCO2, voc, nil
}

func measureBMxx80(bme *bmxx80.Dev) (float64, float64, float64, error) {
	var env physic.Env
	if err := bme.Sense(&env); err != nil {
		return 0, 0, 0, fmt.Errorf("BME: %w", err)
	}
	temp := env.Temperature + physic.Temperature(*tempOffset)*physic.Celsius
	//log.Printf("BME2xx %8s(%8s) %10s %9s ", temp, env.Temperature, env.Pressure, env.Humidity)

	inTemp := float64(temp.Celsius())
	inHumid := float64(env.Humidity) / float64(physic.PercentRH)
	hPaMSL := calcMeanHeightAirPressure(float64(env.Pressure)/float64(physic.Pascal*100), inTemp, *aboveSeaLevel)

	return inTemp, inHumid, hPaMSL, nil
}

func measureCCS811(inTemp, inHumid float64, ccs *ccs811.Dev) (float64, float64, error) {

	if err := ccs.SetEnvironmentData(float32(inTemp), float32(inHumid)); err != nil {
		return 0, 0, fmt.Errorf("CCS init: %w", err)
	}

	var air ccs811.SensorValues
	if err := ccs.SensePartial(ccs811.ReadCO2VOCStatus, &air); err != nil {
		return 0, 0, fmt.Errorf("CCS: %w", err)
	}

	//log.Printf("eCO2:%dppm VOC:%dppb\n", air.ECO2, air.VOC)

	return float64(air.ECO2), float64(air.VOC), nil

}

func measureSHT3x(sht *SHT3x) (float64, float64, error) {
	inTemp, inHumid, err := sht.ReadTemperatureAndRelativeHumidity(sht3x.RepeatabilityMedium)
	if err != nil {
		err = fmt.Errorf("SHT3x: %w", err)
		return 0, 0, err
	}
	//log.Printf("SHT3x %v*C, %v%%\n", inTemp, inHumid)

	return inTemp, inHumid, nil
}
