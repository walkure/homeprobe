package metrics

import (
	"bytes"
	"testing"
)

func TestMetricSet(t *testing.T) {

	s := MetricSet{}
	v1 := NewGauge("testValue1", "testHelp1")
	v2 := NewGauge("testValue2", "testHelp2")
	s.Add(v1)
	s.Add(v2)

	v1.Set(Labels{"1": "b", "2": "a"},
		RoundFloat64{
			Value:     1134.43543,
			Precision: 2,
		})

	v2.Set(Labels{"1": "b", "2": "a"},
		RoundFloat64{
			Value:     1134.43543,
			Precision: 2,
		})


	var buf bytes.Buffer
	err := s.Write(&buf)
	if err != nil {
		t.Errorf("metricSet.Write() failed: %v", err)
	}

	got := buf.String()
	want := `# HELP testValue1 testHelp1
# TYPE testValue1 gauge
testValue1{1="b",2="a"} 1134.44
# HELP testValue2 testHelp2
# TYPE testValue2 gauge
testValue2{1="b",2="a"} 1134.44
`
	if got != want {
		t.Errorf("metricSet.Write() failed: got:%q want:%q", got, want)
	}
}

func TestGaugeMetricNoLabel(t *testing.T) {
	v := NewGauge("testValue", "testHelp")
	v.Set(nil,
		RoundFloat64{
			Value:     1134.43543,
			Precision: 2,
		})

	var buf bytes.Buffer
	err := v.outputMetric(&buf)
	if err != nil {
		t.Errorf("gaugeMetric.outputMetric() failed: %v", err)
	}

	got := buf.String()

	want := `# HELP testValue testHelp
# TYPE testValue gauge
testValue 1134.44
`
	if got != want {
		t.Errorf("gaugeMetric.outputMetric() failed: got:%q want:%q", got, want)
	}
}

func TestGaugeMetric(t *testing.T) {
	v := NewGauge("testValue", "testHelp")
	v.Set(Labels{"1": "b", "2": "a"},
		RoundFloat64{
			Value:     1134.43543,
			Precision: 2,
		})

	var buf bytes.Buffer
	err := v.outputMetric(&buf)
	if err != nil {
		t.Errorf("gaugeMetric.outputMetric() failed: %v", err)
	}

	got := buf.String()

	want := `# HELP testValue testHelp
# TYPE testValue gauge
testValue{1="b",2="a"} 1134.44
`
	if got != want {
		t.Errorf("gaugeMetric.outputMetric() failed: got:%q want:%q", got, want)
	}
}

func TestGaugeMetricMultipleAndUpdate(t *testing.T) {
	v := NewGauge("testValue", "testHelp")
	v.Set(Labels{"1": "b", "2": "a"},
		RoundFloat64{
			Value:     1134.43543,
			Precision: 2,
		})
	v.Set(Labels{"1": "b", "2": "a"},
		RoundFloat64{
			Value:     2134.43543,
			Precision: 2,
		})
	v.Set(Labels{"1": "c", "2": "a"},
		RoundFloat64{
			Value:     3134.43543,
			Precision: 2,
		})
	var buf bytes.Buffer
	err := v.outputMetric(&buf)
	if err != nil {
		t.Errorf("gaugeMetric.outputMetric() failed: %v", err)
	}

	got := buf.String()

	want := `# HELP testValue testHelp
# TYPE testValue gauge
testValue{1="b",2="a"} 2134.44
testValue{1="c",2="a"} 3134.44
`
	if got != want {
		t.Errorf("gaugeMetric.outputMetric() failed: got:%q want:%q", got, want)
	}

}

func TestMetricEntityToString(t *testing.T) {
	v := metricEntity{
		name:       "testName",
		help:       "testHelp",
		metricType: "testType",
		values: map[string]metricValueItem{
			"{1=\"b\",2=\"a\"}": metricStringerItem{
				labels: Labels{"2": "a", "1": "b"},
				value: RoundFloat64{
					Value:     1134.43543,
					Precision: 2,
				},
			},
		},
	}

	want := `# HELP testName testHelp
# TYPE testName testType
testName{1="b",2="a"} 1134.44
`
	var buf bytes.Buffer
	err := v.outputMetric(&buf)
	if err != nil {
		t.Errorf("metricEntity.outputMetric() failed: %v", err)
	}

	got := buf.String()
	if got != want {
		t.Errorf("metricValueSet.outputMetric() failed: got:%q want:%q", got, want)
	}

}

func TestMetricStringerValueItem(t *testing.T) {

	v := metricStringerItem{
		labels: Labels{"2": "a", "1": "b"},
		value: RoundFloat64{
			Value:     1134.43543,
			Precision: 2,
		},
	}


	var buf bytes.Buffer
	err := v.writeValue("metricName", &buf)
	if err != nil {
		t.Errorf("metricStringerItem.writeValue() failed: %v", err)
	}

	got := buf.String()
	want := "metricName{1=\"b\",2=\"a\"} 1134.44\n"

	if got != want {
		t.Errorf("metricSimpleValueItem.Metric() failed: got:%q want:%q", got, want)
	}
}

func TestLabelsString(t *testing.T) {

	v := Labels{"a": "b", "c": "d", "e": "f"}

	got := v.String()
	want := "{a=\"b\",c=\"d\",e=\"f\"}"

	if got != want {
		t.Errorf("labels.String() failed: got:%q want:%q", got, want)
	}
}
