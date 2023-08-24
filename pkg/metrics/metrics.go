package metrics

import (
	"fmt"
	"github.com/walkure/homeprobe/pkg/util"
	"io"
	"strconv"
	"strings"
)

type MetricSet map[string]Metric

func (s MetricSet) Add(m Metric){
	s[m.entityName()] = m
}

func (s MetricSet) Write(w io.Writer) error {
	for _, k := range util.Keys(s) {
		if err := s[k].outputMetric(w); err != nil {
			return err
		}
	}
	return nil
}

func NewGauge(name, help string) Metric {
	return &metricEntity{
		name:       name,
		help:       help,
		values:     make(map[string]metricValueItem),
		metricType: "gauge",
	}
}

type Metric interface {
	entityName() string
	outputMetric(w io.Writer) error
	Set(labels Labels, value RoundFloat64)
}

type metricEntity struct {
	metricType string
	name       string
	help       string
	values     map[string]metricValueItem
}

var noneLabels = make(Labels)

func (m metricEntity) entityName() string {
	return m.metricType + "_" + m.name
}

func (m metricEntity) Set(labels Labels, value RoundFloat64) {
	if labels == nil {
		labels = noneLabels
	}
	m.values[labels.String()] = metricStringerItem{
		labels: labels,
		value:  value,
	}
}

func (m metricEntity) outputMetric(w io.Writer) error {
	if len(m.values) == 0 {
		// No values, no output
		return nil
	}

	io.WriteString(w, fmt.Sprintf("# HELP %s %s\n", m.name, m.help))
	io.WriteString(w, fmt.Sprintf("# TYPE %s %s\n", m.name, m.metricType))
	for _, k := range util.Keys(m.values) {
		m.values[k].writeValue(m.name,w)
	}

	return nil
}

type metricValueItem interface {
	writeValue(name string,w io.Writer) error
}

// metricStringerItem is a stringer metric value with labels
type metricStringerItem struct {
	labels Labels
	value  fmt.Stringer
}

func (m metricStringerItem) writeValue(name string,w io.Writer) error {
	io.WriteString(w, name)
	io.WriteString(w, m.labels.String())
	io.WriteString(w, " ")
	io.WriteString(w, m.value.String())
	io.WriteString(w, "\n")

	return nil
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
