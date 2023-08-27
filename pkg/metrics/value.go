package metrics

import (
	"math"
	"strconv"
)

// RoundFloat64 is a stringer float64 with precision round
type RoundFloat64 struct {
	Value     float64
	Precision int
}

func (v RoundFloat64) String() string {
	shift := math.Pow10(v.Precision)
	round := math.Round(v.Value*shift) / shift
	return strconv.FormatFloat(round, 'f', v.Precision, 64)
}
