package logging

import (
    "time"
"github.com/gin-gonic/gin"
    "github.com/sirupsen/logrus"
)

type Logger struct {
    logger         *logrus.Logger
    service        string
    env            string
    metricsEnabled bool
    metrics        Metrics
}

type Metrics interface {
    RecordCount(metricName string, value float64, tags map[string]string)
    RecordGauge(metricName string, value float64, tags map[string]string)
    RecordTiming(metricName string, duration time.Duration, tags map[string]string)
}

// NewLogger initializes the Logger with metrics capability if enabled
func NewLogger(service, env string, metricsEnabled bool, metrics Metrics) *Logger {
    logger := logrus.New()
    logger.SetFormatter(&logrus.JSONFormatter{})
    logger.SetLevel(logrus.InfoLevel)

    return &Logger{
        logger:         logger,
        service:        service,
        env:            env,
        metricsEnabled: metricsEnabled,
        metrics:        metrics,
    }
}

// LogAndRecord logs the message and records the metric if enabled
func (l *Logger) LogAndRecord(level logrus.Level, msg string, metricName string, tags map[string]string, duration ...time.Duration) {
    entry := l.logger.WithFields(logrus.Fields{
        "service": l.service,
        "env":     l.env,
    })

    for key, value := range tags {
        entry = entry.WithField(key, value)
    }

    entry.Log(level, msg)

    if l.metricsEnabled {
        if len(duration) > 0 {
            l.metrics.RecordTiming(metricName, duration[0], tags)
        } else {
            l.metrics.RecordCount(metricName, 1, tags)
        }
    }
}

// Middleware provides a Gin middleware for logging
func (l *Logger) Middleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        path := c.Request.URL.Path
        method := c.Request.Method

        l.logger.Infof("Incoming request: %s %s", method, path)
        c.Next()

        statusCode := c.Writer.Status()
        l.logger.Infof("Request completed: %s %s - Status: %d", method, path, statusCode)
    }
}

