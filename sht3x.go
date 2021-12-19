package main

import (
	"fmt"
	gi2c "github.com/d2r2/go-i2c"
	sht3x "github.com/d2r2/go-sht3x"
)

type SHT3x struct {
	i2c    *gi2c.I2C
	sensor *sht3x.SHT3X
}

func NewSHT3x(bus int) (*SHT3x, error) {
	v := SHT3x{}
	i2c, err := gi2c.NewI2C(uint8(bus), 1)
	if err != nil {
		return nil, fmt.Errorf("cannot open i2c device:%w", err)
	}
	v.i2c = i2c
	v.sensor = sht3x.NewSHT3X()

	return &v, nil
}

func (v *SHT3x) Reset() error {
	return v.sensor.Reset(v.i2c)
}

func (v *SHT3x) ReadTemperatureAndRelativeHumidity(precision sht3x.MeasureRepeatability) (float64, float64, error) {
	temp, humid, err := v.sensor.ReadTemperatureAndRelativeHumidity(v.i2c, precision)
	return float64(temp), float64(humid), err
}

func (v *SHT3x) Close() {
	v.i2c.Close()
}
