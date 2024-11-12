package logging

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"strconv"
	"time"
	//"go.opentelemetry.io/otel"
	//"go.opentelemetry.io/otel/propagation"
)

type Logger struct {
	logger         *logrus.Logger
	owner          string
	env            string
	logLevel       logrus.Level
	metricsEnabled bool
	metrics        Metrics
	ctx            context.Context
}

type Metrics interface {
	RecordCount(metricName string, value float64, tags map[string]string)
	RecordGauge(metricName string, value float64, tags map[string]string)
	RecordTiming(metricName string, duration time.Duration, tags map[string]string)
}

// NewLogger initializes the Logger with metrics capability if enabled
func NewLogger(owner, env, logLevel string, metricsEnabled bool, metrics Metrics, ctx context.Context) *Logger {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	level, err := logrus.ParseLevel(logLevel)
	if err != nil {
		level = logrus.InfoLevel // Default level
	}
	logger.SetLevel(level)

	return &Logger{
		logger:         logger,
		owner:          owner,
		env:            env,
		logLevel:       level,
		metricsEnabled: metricsEnabled,
		metrics:        metrics,
		ctx:            ctx,
	}
}

// LogWithStats logs the message and records the metric if enabled
func (l *Logger) LogWithStats(logLevel, msg string, metric map[string]string, extra map[string]interface{}) {
	level, err := logrus.ParseLevel(logLevel)
	if err != nil {
		level = logrus.InfoLevel // Default level
	}
	entry := l.logger.WithFields(logrus.Fields{
		"owner": l.owner,
		"env":   l.env,
	})

	for key, value := range metric {
		entry = entry.WithField(key, value)
	}
	for key, value := range extra {
		entry = entry.WithField(key, value)
	}

	entry.Log(level, msg)

	// Process metrics if enabled and metric map is provided
	if l.metricsEnabled && metric != nil {
		metricName, ok := metric["metric_name"]
		if !ok {
			metricName = "default_metric" // Provide a fallback metric name if missing
		}

		metricType, ok := metric["metric_type"]
		if !ok {
			metricType = "count" // Default to count if type is not specified
		}

		metricValue := 1.0 // Default value
		if val, ok := metric["metric_value"]; ok {
			metricValue = parseMetricValue(val)
		}

		// Remove control fields to use the rest as tags
		delete(metric, "metric_name")
		delete(metric, "metric_value")
		delete(metric, "metric_type")

		tags := make(map[string]string)
		for k, v := range metric {
			tags[k] = v
		}

		// Record metrics based on type
		switch metricType {
		case "count":
			l.metrics.RecordCount(metricName, metricValue, tags)
		case "gauge":
			l.metrics.RecordGauge(metricName, metricValue, tags)
		case "timing":
			if len(extra) > 0 {
				if duration, ok := extra["duration"].(time.Duration); ok {
					l.metrics.RecordTiming(metricName, duration, tags)
				}
			}
		}
	}
}

// Middleware provides a Gin middleware for logging
func (l *Logger) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		latency := time.Since(start)

		statusCode := c.Writer.Status()
		l.logger.WithFields(logrus.Fields{
			"status_code": statusCode,
			"latency":     latency,
			"client_ip":   c.ClientIP(),
			"method":      c.Request.Method,
			"path":        c.Request.URL.Path,
		}).Info("request handled")
	}
}

// Helper function to parse metric value from string
func parseMetricValue(val string) float64 {
	value, err := strconv.ParseFloat(val, 64)
	if err != nil {
		value = 1.0 // Default to 1 if parsing fails
	}
	return value
}
