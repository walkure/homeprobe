package main

import (
	"io"
	"maps"
	"time"

	"github.com/walkure/homeprobe/pkg/metrics"
)

type MetricData struct {
	temp            metrics.Metric
	relHumid        metrics.Metric
	absHumid        metrics.Metric
	disconfortIndex metrics.Metric
	vBattery        metrics.Metric
	ttl             time.Duration
	baseLabels      metrics.Labels
	d               metrics.MetricSet
}

func NewMetrics(ttl time.Duration, baseLabels metrics.Labels) *MetricData {
	m := &MetricData{
		temp:            metrics.NewGauge("temperature", "Temperature"),
		relHumid:        metrics.NewGauge("relative_humidity", "Relative Humidity percent"),
		absHumid:        metrics.NewGauge("absolute_humidity", "Absolute Humidity g/m^3"),
		disconfortIndex: metrics.NewGauge("disconfort_index", "Disconfort Index"),
		vBattery:        metrics.NewGauge("sensor_vbat", "Voltage of Sensor battery"),
		ttl:             ttl,
		baseLabels:      baseLabels,
	}

	d := metrics.MetricSet{}
	d.Add(m.temp, m.relHumid, m.absHumid, m.disconfortIndex, m.vBattery)
	m.d = d

	return m
}

func (m *MetricData) Write(w io.Writer) error {
	return m.d.Write(w)
}

func mergeLabels(base, extra metrics.Labels) metrics.Labels {
	if extra == nil {
		return base
	}

	ret := maps.Clone(base)
	for k, v := range extra {
		ret[k] = v
	}
	return ret
}

func (m *MetricData) UpdateTemperature(value float64, extra metrics.Labels) {
	m.temp.SetWithTimeout(
		mergeLabels(m.baseLabels, extra),
		metrics.RoundFloat64{
			Value:     value,
			Precision: 2,
		},
		time.Now().Add(m.ttl),
	)
}

func (m *MetricData) UpdateRelativeHumidity(value float64, extra metrics.Labels) {
	m.relHumid.SetWithTimeout(
		mergeLabels(m.baseLabels, extra),
		metrics.RoundFloat64{
			Value:     value,
			Precision: 0,
		},
		time.Now().Add(m.ttl),
	)
}

func (m *MetricData) UpdateAbsoluteHumidity(value float64, extra metrics.Labels) {
	m.absHumid.SetWithTimeout(
		mergeLabels(m.baseLabels, extra),
		metrics.RoundFloat64{
			Value:     value,
			Precision: 2,
		},
		time.Now().Add(m.ttl),
	)
}

func (m *MetricData) UpdateDisconfortIndex(value float64, extra metrics.Labels) {
	m.disconfortIndex.SetWithTimeout(
		mergeLabels(m.baseLabels, extra),
		metrics.RoundFloat64{
			Value:     value,
			Precision: 2,
		},
		time.Now().Add(m.ttl),
	)
}

func (m *MetricData) UpdateBattery(value uint8, extra metrics.Labels) bool {
	if value > 100 {
		return false
	}

	m.vBattery.SetWithTimeout(
		mergeLabels(m.baseLabels, extra),
		metrics.RoundFloat64{
			Value:     float64(value) / 100.0 * 3,
			Precision: 3,
		},
		time.Now().Add(m.ttl),
	)

	return true
}
