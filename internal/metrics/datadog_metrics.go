package metrics

import (
	"fmt"
	"time"

	"github.com/DataDog/datadog-go/statsd"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	ddotel "gopkg.in/DataDog/dd-trace-go.v1/ddtrace/opentelemetry"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

// DataDogMetrics handles metrics with Datadog and sets up OpenTelemetry tracing.
type DataDogMetrics struct {
	client         *statsd.Client
	tracerProvider trace.TracerProvider
}

// NewDataDogMetrics initializes Datadog metrics with OpenTelemetry tracing.
func NewDataDogMetrics(ddAgentAddr, serviceName, env string) (*DataDogMetrics, error) {
	// Initialize statsd client for Datadog metrics
	client, err := statsd.New(fmt.Sprintf("%s:%s", ddAgentAddr, "8125"))
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Datadog metrics client: %w", err)
	}

	// Initialize the Datadog OpenTelemetry tracer provider
	//tracerProvider := ddotel.NewTracerProvider()
	tracerProvider := ddotel.NewTracerProvider(
		tracer.WithAgentAddr(fmt.Sprintf("%s:%s", ddAgentAddr, "8126")),
		tracer.WithService(serviceName),
		tracer.WithEnv(env),
		tracer.WithGlobalTag("env", env),
		tracer.WithGlobalTag("service", serviceName),
	)
	otel.SetTracerProvider(tracerProvider)

	return &DataDogMetrics{
		client:         client,
		tracerProvider: tracerProvider,
	}, nil
}

// Close gracefully shuts down the tracer provider and closes the metrics client.
func (m *DataDogMetrics) Close() error {
	if provider, ok := m.tracerProvider.(*ddotel.TracerProvider); ok {
		provider.Shutdown()
	}
	if m.client != nil {
		return m.client.Close()
	}
	return nil
}

// RecordCount records a count metric in Datadog.
func (m *DataDogMetrics) RecordCount(metricName string, value float64, tags map[string]string) {
	tagList := formatTags(tags)
	_ = m.client.Count(metricName, int64(value), tagList, 1)
}

// RecordGauge records a gauge metric in Datadog.
func (m *DataDogMetrics) RecordGauge(metricName string, value float64, tags map[string]string) {
	tagList := formatTags(tags)
	_ = m.client.Gauge(metricName, value, tagList, 1)
}

// RecordTiming records a timing metric in Datadog.
func (m *DataDogMetrics) RecordTiming(metricName string, duration time.Duration, tags map[string]string) {
	tagList := formatTags(tags)
	_ = m.client.Timing(metricName, duration, tagList, 1)
}

// formatTags converts a map of tags into a slice of strings for Datadog.
func formatTags(tags map[string]string) []string {
	tagList := []string{}
	for k, v := range tags {
		tagList = append(tagList, fmt.Sprintf("%s:%s", k, v))
	}
	return tagList
}
