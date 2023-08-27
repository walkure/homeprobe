package metrics

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/walkure/homeprobe/pkg/util"
)

type MetricSet map[string]Metric

func (s MetricSet) Add(m Metric) {
	s[m.entityName()] = m
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
	if len(m.values) == 0 {
		// No values, no output
		return nil
	}

	// check timeout
	for _, k := range m.values {
		if it, ok := k.(metricExpirableItem); ok {
			if ok, label := it.expired(now); ok {
				delete(m.values, label)
			}
		}
	}

	io.WriteString(w, fmt.Sprintf("# HELP %s %s\n", m.metricName, m.help))
	io.WriteString(w, fmt.Sprintf("# TYPE %s %s\n", m.metricName, m.metricType))
	for _, k := range util.Keys(m.values) {
		m.values[k].writeValue(m.metricName, w)
	}

	return nil
}

type metricValueItem interface {
	writeValue(name string, w io.Writer) error
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
