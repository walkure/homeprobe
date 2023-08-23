package main

import (
	"math"
	"strconv"
)

func calcAbsoluteHumidity(temp, relativeHumid float64) float64 {
	vaporPressureSat := 6.1078 * math.Pow(10, 7.5*temp/(temp+237.7))
	vaporAmountSat := 217 * vaporPressureSat / (temp + 273.15)

	return vaporAmountSat * relativeHumid / 100
}

func calcDisconfortIndex(temp, relativeHumid float64) float64 {
	return 0.81*temp + 0.01*relativeHumid*(0.99*temp-14.3) + 46.3
}

func roundFormat(value float64, places int) string {
	shift := math.Pow10(places)
	round := math.Round(value*shift) / shift
	return strconv.FormatFloat(round, 'f', places, 64)
}

func calcMeanHeightAirPressure(pressure, temp, height float64) float64 {
	if height <= 0 {
		return pressure
	}

	kelvin := temp + 273.15
	return pressure * math.Pow(kelvin/(kelvin+0.0065*height), -5.257)
}
