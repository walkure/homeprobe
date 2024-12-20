package metrics

import (
	"fmt"
	"io"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	loggerFactory "github.com/walkure/homeprobe/pkg/logger"
	"github.com/walkure/homeprobe/pkg/util"
)

type MetricSet map[string]Metric

func (s MetricSet) Add(m ...Metric) {
	for _, it := range m {
		s[it.entityName()] = it
	}
}

func (s MetricSet) Write(w io.Writer) error {
	now := time.Now()
	for _, k := range util.Keys(s) {
		if err := s[k].outputMetric(w, now); err != nil {
			return err
		}
	}
	return nil
}

// satisfy slog.LogValuer interface
func (s MetricSet) LogValue() slog.Value {

	v := []slog.Attr{}
	for _, k := range util.Keys(s) {
		v = append(v, s[k].LogAttr())
	}

	return slog.GroupValue(v...)
}

func NewGauge(name, help string) Metric {
	return &metricEntity{
		metricName: name,
		help:       help,
		values:     make(map[string]metricValueItem),
		metricType: "gauge",
	}
}

type Metric interface {
	entityName() string
	outputMetric(w io.Writer, now time.Time) error
	Set(labels Labels, value RoundFloat64)
	SetWithTimeout(labels Labels, value RoundFloat64, expireAt time.Time)
	LogAttr() slog.Attr
}

type metricEntity struct {
	metricType string
	metricName string
	help       string
	values     map[string]metricValueItem
	mu         sync.Mutex
}

var noneLabels = make(Labels)

func (m *metricEntity) entityName() string {
	return m.metricType + "_" + m.metricName
}

func (m *metricEntity) Set(labels Labels, value RoundFloat64) {
	m.SetWithTimeout(labels, value, time.Time{})
}

func (m *metricEntity) SetWithTimeout(labels Labels, value RoundFloat64, expireAt time.Time) {
	if labels == nil {
		labels = noneLabels
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.values[labels.String()] = metricStringerItem{
		labels:   labels,
		value:    value,
		expireAt: expireAt,
	}
}

func (m *metricEntity) outputMetric(w io.Writer, now time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	logger := loggerFactory.GetLogger("metrics")

	// check timeout
	for _, k := range m.values {
		if it, ok := k.(metricExpirableItem); ok {
			if ok, label := it.expired(now); ok {
				logger.Warn("expired metrics deleted",
					slog.String("metric", m.metricName),
					slog.String("label", label),
					slog.String("value", m.values[label].valueToString()),
				)
				delete(m.values, label)
			}
		}
	}

	if len(m.values) == 0 {
		// No values, no output
		return nil
	}

	io.WriteString(w, fmt.Sprintf("# HELP %s %s\n", m.metricName, m.help))
	io.WriteString(w, fmt.Sprintf("# TYPE %s %s\n", m.metricName, m.metricType))
	for _, k := range util.Keys(m.values) {
		m.values[k].writeValue(m.metricName, w)
	}

	return nil
}

func (m *metricEntity) LogAttr() slog.Attr {
	v := []slog.Attr{}
	for _, k := range util.Keys(m.values) {
		v = append(v, m.values[k].logAttr())
	}
	return slog.Group(m.metricName, "", v)
}

type metricValueItem interface {
	writeValue(name string, w io.Writer) error
	valueToString() string
	logAttr() slog.Attr
}

type metricExpirableItem interface {
	expired(now time.Time) (bool, string)
}

// metricStringerItem is a stringer metric value with labels
type metricStringerItem struct {
	labels   Labels
	value    fmt.Stringer
	expireAt time.Time
}

func (m metricStringerItem) writeValue(name string, w io.Writer) error {
	io.WriteString(w, name)
	io.WriteString(w, m.labels.String())
	io.WriteString(w, " ")
	io.WriteString(w, m.value.String())
	io.WriteString(w, "\n")

	return nil
}

func (m metricStringerItem) valueToString() string {
	return m.value.String()
}

func (m metricStringerItem) expired(now time.Time) (bool, string) {
	if m.expireAt.IsZero() {
		return false, ""
	}

	// now >= expireAt
	if !now.Before(m.expireAt) {
		return true, m.labels.String()
	}

	return false, ""
}

func (m metricStringerItem) logAttr() slog.Attr {
	return slog.Group("metric",
		m.labels.LogAttr(),
		slog.String("value", m.value.String()),
	)
}

// Labels is a set of labels for a metric
type Labels map[string]string

func (l Labels) String() string {
	if len(l) == 0 {
		return ""
	}

	kvs := make([]string, 0, len(l))
	for _, k := range util.Keys(l) {
		kvs = append(kvs, fmt.Sprintf("%s=%s", k, strconv.Quote(l[k])))
	}
	return "{" + strings.Join(kvs, ",") + "}"
}

func (l Labels) LogAttr() slog.Attr {

	a := []slog.Attr{}
	for _, k := range util.Keys(l) {
		a = append(a, slog.String(k, l[k]))
	}

	return slog.Group("labels", "", a)
}
