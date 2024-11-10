package metrics

import (
	"fmt"
	"github.com/DataDog/datadog-go/statsd"
	"time"
)

type DataDogMetrics struct {
	client *statsd.Client
}

// NewDataDogMetrics initializes a new DataDogMetrics instance
func NewDataDogMetrics(addr string) (*DataDogMetrics, error) {
	client, err := statsd.New(addr)
	if err != nil {
		return nil, err
	}
	return &DataDogMetrics{client: client}, nil
}

func (m *DataDogMetrics) RecordCount(metricName string, value float64, tags map[string]string) {
	tagList := formatTags(tags)
	_ = m.client.Count(metricName, int64(value), tagList, 1)
}

func (m *DataDogMetrics) RecordGauge(metricName string, value float64, tags map[string]string) {
	tagList := formatTags(tags)
	_ = m.client.Gauge(metricName, value, tagList, 1)
}

func (m *DataDogMetrics) RecordTiming(metricName string, duration time.Duration, tags map[string]string) {
	tagList := formatTags(tags)
	_ = m.client.Timing(metricName, duration, tagList, 1)
}

func formatTags(tags map[string]string) []string {
	tagList := []string{}
	for k, v := range tags {
		tagList = append(tagList, fmt.Sprintf("%s:%s", k, v))
	}
	return tagList
}
