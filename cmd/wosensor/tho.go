package main

import (
	"log/slog"
	"sync"

	"github.com/walkure/gatt"
	"github.com/walkure/go-wosensors"
	loggerFactory "github.com/walkure/homeprobe/pkg/logger"
	"github.com/walkure/homeprobe/pkg/metrics"
	"github.com/walkure/homeprobe/pkg/weather"
)

type THO struct {
	deviceId string
	logger   *slog.Logger
	mu       sync.Mutex
	seqno    uint8
	m        *MetricData
}

func NewTHO(deviceId string, d *MetricData) *THO {

	logger := loggerFactory.GetLogger("tho")

	if deviceId == "" {
		logger.Warn("deviceId is mandatory. Disabled")
		return nil
	}

	return &THO{
		deviceId: deviceId,
		logger:   logger,
		m:        d,
	}
}

func (t *THO) Handler(next func(gatt.Peripheral, *gatt.Advertisement, int)) func(gatt.Peripheral, *gatt.Advertisement, int) {
	if t == nil {
		return next
	}

	labels := metrics.Labels{"wosensor_id": t.deviceId, "wosensor_type": "tho"}
	handler := func(d wosensors.THOData) {
		t.mu.Lock()
		defer t.mu.Unlock()

		if t.seqno == d.SequenceNumber {
			scanType := "active"
			if d.BatteryPercent > 100 {
				scanType = "passive"
			}

			t.logger.Debug("sequence not changed",
				slog.Uint64("seq", uint64(d.SequenceNumber)),
				slog.String("scan_type", scanType),
			)
			return
		}

		t.seqno = d.SequenceNumber

		t.logger.Debug("new data", "d", d, "seq", d.SequenceNumber)

		t.m.UpdateTemperature(float64(d.Temperature), labels)
		t.m.UpdateRelativeHumidity(float64(d.Humidity), labels)
		t.m.UpdateAbsoluteHumidity(weather.AbsoluteHumidity(float64(d.Temperature), float64(d.Humidity)), labels)
		t.m.UpdateDisconfortIndex(weather.DisconfortIndex(float64(d.Temperature), float64(d.Humidity)), labels)
		if d.BatteryPercent <= 100 {
			t.m.UpdateBattery(float64(d.BatteryPercent)/100.0*3, labels)
		}

	}

	// active scanning only
	return wosensors.HandleWoSensorTHO(*woSensorTHOId, false, handler, next)
}
