package main

import (
	"context"
	"fmt"

	//"log"

	"periph.io/x/conn/v3/physic"
	"periph.io/x/devices/v3/bmxx80"
	"periph.io/x/devices/v3/ccs811"

	"github.com/walkure/go-lpsensors"
	"github.com/walkure/homeprobe/pkg/metrics"
	"github.com/walkure/homeprobe/pkg/weather"

	sht3x "github.com/d2r2/go-sht3x"
)

func measure(bme *bmxx80.Dev, ccs *ccs811.Dev, sht *SHT3x, lps *lpsensors.Dev) (metrics.MetricSet, error) {

	var inTemp, inHumid, hPaMSL float64
	var err error

	s := metrics.MetricSet{}
	temperature := metrics.NewGauge("temperature", "Temperature")
	relativeHumidity := metrics.NewGauge("relative_humidity", "Relative Humidity percent")
	absoluteHumidity := metrics.NewGauge("absolute_humidity", "Absolute Humidity g/m3")
	disconfortIndex := metrics.NewGauge("disconfort_index", "Disconfort Index")
	airPressure := metrics.NewGauge("pressure", "Air Pressure hPa")
	eCO2ppm := metrics.NewGauge("eco2", "eCO2 ppm")
	vocppb := metrics.NewGauge("voc", "VOC ppb")

	s.Add(temperature)
	s.Add(relativeHumidity)
	s.Add(absoluteHumidity)
	s.Add(disconfortIndex)
	s.Add(airPressure)
	s.Add(eCO2ppm)
	s.Add(vocppb)

	labels := metrics.Labels{"place": "inside"}

	if bme != nil {
		inTemp, inHumid, hPaMSL, err = measureBMxx80(bme)
		if err != nil {
			return nil, err
		}
	}

	if sht != nil {
		inTemp, inHumid, err = measureSHT3x(sht)
		if err != nil {
			return nil, err
		}
	}

	if lps != nil {
		inTemp, hPaMSL, err = measureLPS(lps)
		if err != nil {
			return nil, err
		}
	}

	temperature.Set(
		labels,
		metrics.RoundFloat64{
			Value:     inTemp,
			Precision: 2,
		},
	)

	if bme != nil || lps != nil {
		airPressure.Set(
			labels,
			metrics.RoundFloat64{
				Value:     hPaMSL,
				Precision: 2,
			},
		)
	}

	if bme != nil || sht != nil {
		relativeHumidity.Set(
			labels,
			metrics.RoundFloat64{
				Value:     inHumid,
				Precision: 2,
			},
		)

		absoluteHumidity.Set(
			labels,
			metrics.RoundFloat64{
				Value:     weather.AbsoluteHumidity(inTemp, inHumid),
				Precision: 2,
			},
		)

		disconfortIndex.Set(
			labels,
			metrics.RoundFloat64{
				Value:     weather.DisconfortIndex(inTemp, inHumid),
				Precision: 2,
			},
		)
	}

	if ccs != nil {
		var eCO2, voc float64
		eCO2, voc, err = measureCCS811(inTemp, inHumid, ccs)
		if err != nil {
			return nil, err
		}
		eCO2ppm.Set(
			labels,
			metrics.RoundFloat64{
				Value:     eCO2,
				Precision: 2,
			},
		)

		vocppb.Set(
			labels,
			metrics.RoundFloat64{
				Value:     voc,
				Precision: 2,
			},
		)
	}

	return s, nil
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
	hPaMSL := weather.MeanHeightAirPressure(float64(env.Pressure)/float64(physic.Pascal*100), inTemp, *aboveSeaLevel)

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

func measureLPS(lps *lpsensors.Dev) (float64, float64, error) {
	var env lpsensors.SensorValues
	if err := lps.Sense(context.Background(), &env); err != nil {
		return 0, 0, fmt.Errorf("LPS: %w", err)
	}
	//log.Printf("LPS331AP %s\n", env.String())

	inTemp := float64(env.Temperature.Celsius())
	hPaMSL := weather.MeanHeightAirPressure(float64(env.Pressure)/float64(physic.Pascal*100), inTemp, *aboveSeaLevel)

	return inTemp, hPaMSL, nil
}
