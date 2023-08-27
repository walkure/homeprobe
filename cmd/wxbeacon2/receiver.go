package main

import (
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/walkure/go-wxbeacon2"
	"github.com/walkure/homeprobe/pkg/metrics"
	"github.com/walkure/homeprobe/pkg/weather"
)

var lastSeqID = atomic.Uint32{}

type envData struct {
	temp            metrics.Metric
	relHumid        metrics.Metric
	absHumid        metrics.Metric
	ambientLight    metrics.Metric
	uvIndex         metrics.Metric
	pressure        metrics.Metric
	soundNoise      metrics.Metric
	disconfortIndex metrics.Metric
	heatStoke       metrics.Metric
	vBattery        metrics.Metric
}

var wxbeaconData *envData

func initEnvData() metrics.MetricSet {

	wxbeaconData = &envData{
		temp:            metrics.NewGauge("temperature", "Temperature"),
		relHumid:        metrics.NewGauge("relative_humidity", "Relative Humidity percent"),
		absHumid:        metrics.NewGauge("absolute_humidity", "Absolute Humidity g/m^3"),
		ambientLight:    metrics.NewGauge("ambient_light", "Ambient Light lx"),
		uvIndex:         metrics.NewGauge("uv_index", "Index of UV"),
		pressure:        metrics.NewGauge("pressure", "Pressure hPa"),
		soundNoise:      metrics.NewGauge("sound_noise", "Sound Noise db"),
		disconfortIndex: metrics.NewGauge("disconfort_index", "Disconfort Index"),
		heatStoke:       metrics.NewGauge("heat_stroke", "WGBT"),
		vBattery:        metrics.NewGauge("sensor_vbat", "Voltage of Sensor battery"),
	}

	s := metrics.MetricSet{}
	s.Add(wxbeaconData.absHumid, wxbeaconData.ambientLight, wxbeaconData.disconfortIndex,
		wxbeaconData.heatStoke, wxbeaconData.pressure, wxbeaconData.relHumid, wxbeaconData.soundNoise,
		wxbeaconData.temp, wxbeaconData.uvIndex, wxbeaconData.vBattery)

	return s
}

func wxDataCallback(obj interface{}) {

	if wxbeaconData == nil {
		panic("invoke callback before initialize.")
	}

	data, ok := obj.(wxbeacon2.WxEPData)
	if !ok {
		panic(fmt.Sprintf("WxBeacon2 not EP mode:%T", obj))
	}

	if lastSeqID.CompareAndSwap(uint32(data.Sequence), uint32(data.Sequence)) {
		// sequence not changed.
		return
	}
	lastSeqID.Store(uint32(data.Sequence))
	logger.Info("received", slog.Any("data", data))
	wxbeaconData.setData(data)
}

func (m *envData) setData(data wxbeacon2.WxEPData) {

	labels := metrics.Labels{"place": "outside"}
	// TTL: 15 mins
	expireAt := time.Now().Add(15 * time.Minute)

	m.temp.SetWithTimeout(
		labels,
		metrics.RoundFloat64{
			Value:     data.Temp,
			Precision: 2,
		},
		expireAt,
	)

	m.relHumid.SetWithTimeout(
		labels,
		metrics.RoundFloat64{
			Value:     data.Humid,
			Precision: 2,
		},
		expireAt,
	)

	m.absHumid.SetWithTimeout(
		labels,
		metrics.RoundFloat64{
			Value:     weather.AbsoluteHumidity(data.Temp, data.Humid),
			Precision: 2,
		},
		expireAt,
	)

	m.ambientLight.SetWithTimeout(
		labels,
		metrics.RoundFloat64{
			Value:     float64(data.AmbientLight),
			Precision: 2,
		},
		expireAt,
	)

	m.uvIndex.SetWithTimeout(
		labels,
		metrics.RoundFloat64{
			Value:     float64(data.UVIndex),
			Precision: 2,
		},
		expireAt,
	)

	m.pressure.SetWithTimeout(
		labels,
		metrics.RoundFloat64{
			Value:     weather.MeanHeightAirPressure(data.Pressure, data.Temp, *aboveSeaLevel),
			Precision: 2,
		},
		expireAt,
	)

	m.soundNoise.SetWithTimeout(
		labels,
		metrics.RoundFloat64{
			Value:     data.SoundNoise,
			Precision: 2,
		},
		expireAt,
	)

	m.disconfortIndex.SetWithTimeout(
		labels,
		metrics.RoundFloat64{
			Value:     data.DisconfortIndex,
			Precision: 2,
		},
		expireAt,
	)

	m.heatStoke.SetWithTimeout(
		labels,
		metrics.RoundFloat64{
			Value:     data.HeatStroke,
			Precision: 2,
		},
		expireAt,
	)

	m.vBattery.SetWithTimeout(
		labels,
		metrics.RoundFloat64{
			Value:     data.VBattery,
			Precision: 2,
		},
		expireAt,
	)
}
