// Copyright 2026 PolitePixels Limited
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This project stands against fascism, authoritarianism, and all forms of
// oppression. We built this to empower people, not to enable those who would
// strip others of their rights and dignity.

package logger_otel_sdk

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/monitoring/monitoring_domain"
	"piko.sh/piko/wdk/clock"
	"piko.sh/piko/wdk/logger"
)

const (
	// DefaultMetricsCollectionInterval is the default time between metrics
	// collection cycles.
	DefaultMetricsCollectionInterval = 5 * time.Second

	// errReaderNotRegistered is the error message returned by OTEL's ManualReader
	// when Collect is called before the reader is registered with a MeterProvider.
	// This is normal during startup and should not be logged as a warning.
	errReaderNotRegistered = "reader is not registered"

	// metricTypeCounter is the type label for counter metrics.
	metricTypeCounter = "counter"

	// metricTypeGauge is the metric type identifier for gauge metrics.
	metricTypeGauge = "gauge"

	// metricTypeHistogram is the metric type identifier for histogram metrics.
	metricTypeHistogram = "histogram"
)

// MetricsCollectorOption configures a MetricsCollector.
type MetricsCollectorOption func(*MetricsCollector)

var _ monitoring_domain.MetricsCollectorAdapter = (*MetricsCollector)(nil)

// MetricsCollector wraps the SDK's ManualReader and collects metrics into a
// TelemetryStore for gRPC access. It implements the MetricsCollectorAdapter,
// handlerShutdown, and contextShutdown interfaces.
type MetricsCollector struct {
	// store holds the telemetry storage backend for recording metrics.
	store *monitoring_domain.TelemetryStore

	// manualReader collects metrics on demand; passed to MeterProvider via Reader().
	manualReader *metric.ManualReader

	// clock provides time operations for ticker creation.
	clock clock.Clock

	// stopCh signals when to stop periodic collection.
	stopCh chan struct{}

	// interval is the time between metric collection cycles.
	interval time.Duration
}

// NewMetricsCollector creates a new metrics collector that wraps a ManualReader.
//
// Takes store (*monitoring_domain.TelemetryStore) where metrics will be stored.
// Takes interval (time.Duration) for periodic collection.
//
// Returns *MetricsCollector ready to be used.
func NewMetricsCollector(store *monitoring_domain.TelemetryStore, interval time.Duration, opts ...MetricsCollectorOption) *MetricsCollector {
	if interval <= 0 {
		interval = DefaultMetricsCollectionInterval
	}
	c := &MetricsCollector{
		store:        store,
		manualReader: metric.NewManualReader(),
		stopCh:       make(chan struct{}),
		interval:     interval,
		clock:        clock.RealClock(),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Reader returns the underlying ManualReader which implements
// monitoring_domain.MetricReader. This is what should be passed to the
// MeterProvider.
//
// Returns monitoring_domain.MetricReader which provides access to collected
// metrics.
func (c *MetricsCollector) Reader() monitoring_domain.MetricReader {
	return c.manualReader
}

// Start begins periodic metric collection.
//
// Safe for concurrent use. The spawned goroutine runs until the context is
// cancelled or Stop is called.
func (c *MetricsCollector) Start(ctx context.Context) {
	ctx, l := logger_domain.From(ctx, log)
	go func() {
		ticker := c.clock.NewTicker(c.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C():
				if err := c.collect(ctx); err != nil {
					l.Warn("Failed to collect metrics", logger.Error(err))
				}
			case <-c.stopCh:
				return
			case <-ctx.Done():
				return
			}
		}
	}()
}

// Stop halts the periodic collection of metrics.
func (c *MetricsCollector) Stop() {
	close(c.stopCh)
}

// Shutdown shuts down the underlying ManualReader.
//
// Returns error when the shutdown fails.
func (c *MetricsCollector) Shutdown(ctx context.Context) error {
	c.Stop()
	return c.manualReader.Shutdown(ctx)
}

// collect gathers metrics from the ManualReader and stores them.
//
// Returns error when collecting metrics from the ManualReader fails.
// Returns nil if the reader is not yet registered (expected during startup).
func (c *MetricsCollector) collect(ctx context.Context) error {
	var rm metricdata.ResourceMetrics
	if err := c.manualReader.Collect(ctx, &rm); err != nil {
		if strings.Contains(err.Error(), errReaderNotRegistered) {
			return nil
		}
		return fmt.Errorf("collecting metrics from manual reader: %w", err)
	}

	for _, scopeMetric := range rm.ScopeMetrics {
		for _, m := range scopeMetric.Metrics {
			c.recordMetric(m)
		}
	}

	return nil
}

// recordMetric stores a metric by dispatching to the appropriate handler
// based on its data type.
//
// Takes m (metricdata.Metrics) which is the metric to record.
//
// revive:disable:cyclomatic High case count is required for OTEL metric type
// dispatch
func (c *MetricsCollector) recordMetric(m metricdata.Metrics) {
	switch data := m.Data.(type) {
	case metricdata.Sum[int64]:
		c.recordSumInt64(m, data)
	case metricdata.Sum[float64]:
		c.recordSumFloat64(m, data)
	case metricdata.Gauge[int64]:
		c.recordGaugeInt64(m, data)
	case metricdata.Gauge[float64]:
		c.recordGaugeFloat64(m, data)
	case metricdata.Histogram[int64]:
		c.recordHistogramInt64(m, data)
	case metricdata.Histogram[float64]:
		c.recordHistogramFloat64(m, data)
	}
}

//revive:enable:cyclomatic

// recordSumInt64 records integer sum metric data points to the store.
//
// Takes m (metricdata.Metrics) which provides the metric name, description,
// and unit.
// Takes data (metricdata.Sum[int64]) which contains the integer data points
// to record.
func (c *MetricsCollector) recordSumInt64(m metricdata.Metrics, data metricdata.Sum[int64]) {
	for _, dp := range data.DataPoints {
		attrs := attributeSetToMap(dp.Attributes)
		c.store.RecordMetric(m.Name, m.Description, string(m.Unit), metricTypeCounter, float64(dp.Value), attrs)
	}
}

// recordSumFloat64 records float64 sum metric data points to the store.
//
// Takes m (metricdata.Metrics) which provides the metric name, description,
// and unit.
// Takes data (metricdata.Sum[float64]) which contains the data points to
// record.
func (c *MetricsCollector) recordSumFloat64(m metricdata.Metrics, data metricdata.Sum[float64]) {
	for _, dp := range data.DataPoints {
		attrs := attributeSetToMap(dp.Attributes)
		c.store.RecordMetric(m.Name, m.Description, string(m.Unit), metricTypeCounter, dp.Value, attrs)
	}
}

// recordGaugeInt64 records integer gauge data points to the metrics store.
//
// Takes m (metricdata.Metrics) which provides the metric name, description,
// and unit.
// Takes data (metricdata.Gauge[int64]) which contains the gauge data points
// to record.
func (c *MetricsCollector) recordGaugeInt64(m metricdata.Metrics, data metricdata.Gauge[int64]) {
	for _, dp := range data.DataPoints {
		attrs := attributeSetToMap(dp.Attributes)
		c.store.RecordMetric(m.Name, m.Description, string(m.Unit), metricTypeGauge, float64(dp.Value), attrs)
	}
}

// recordGaugeFloat64 records float64 gauge data points to the metric store.
//
// Takes m (metricdata.Metrics) which provides the metric name, description and
// unit.
// Takes data (metricdata.Gauge[float64]) which contains the gauge data points
// to record.
func (c *MetricsCollector) recordGaugeFloat64(m metricdata.Metrics, data metricdata.Gauge[float64]) {
	for _, dp := range data.DataPoints {
		attrs := attributeSetToMap(dp.Attributes)
		c.store.RecordMetric(m.Name, m.Description, string(m.Unit), metricTypeGauge, dp.Value, attrs)
	}
}

// recordHistogramInt64 records histogram data points with int64 values to the
// metrics store.
//
// Takes m (metricdata.Metrics) which provides the metric name, description,
// and unit.
// Takes data (metricdata.Histogram[int64]) which contains the histogram data
// points to record.
func (c *MetricsCollector) recordHistogramInt64(m metricdata.Metrics, data metricdata.Histogram[int64]) {
	for _, dp := range data.DataPoints {
		attrs := attributeSetToMap(dp.Attributes)
		mean := histogramMeanInt64(dp)
		c.store.RecordMetric(m.Name, m.Description, string(m.Unit), metricTypeHistogram, mean, attrs)
	}
}

// recordHistogramFloat64 records histogram data points with float64 values.
//
// Takes m (metricdata.Metrics) which provides the metric name, description,
// and unit.
// Takes data (metricdata.Histogram[float64]) which contains the histogram
// data points to record.
func (c *MetricsCollector) recordHistogramFloat64(m metricdata.Metrics, data metricdata.Histogram[float64]) {
	for _, dp := range data.DataPoints {
		attrs := attributeSetToMap(dp.Attributes)
		mean := histogramMeanFloat64(dp)
		c.store.RecordMetric(m.Name, m.Description, string(m.Unit), metricTypeHistogram, mean, attrs)
	}
}

// WithMetricsCollectorClock sets the clock used for ticker creation. If not
// provided, the real system clock is used.
//
// Takes clk (clock.Clock) which provides time operations.
//
// Returns MetricsCollectorOption which configures the collector's clock.
func WithMetricsCollectorClock(clk clock.Clock) MetricsCollectorOption {
	return func(c *MetricsCollector) {
		if clk != nil {
			c.clock = clk
		}
	}
}

// histogramMeanInt64 calculates the mean value for an int64 histogram data
// point.
//
// Takes dp (metricdata.HistogramDataPoint[int64]) which is the histogram data
// point to calculate the mean from.
//
// Returns float64 which is the mean value, or zero if the count is zero.
func histogramMeanInt64(dp metricdata.HistogramDataPoint[int64]) float64 {
	if dp.Count > 0 {
		return float64(dp.Sum) / float64(dp.Count)
	}
	return 0
}

// histogramMeanFloat64 calculates the mean value for a float64 histogram data
// point.
//
// Takes dp (metricdata.HistogramDataPoint[float64]) which is the histogram data
// point to calculate the mean from.
//
// Returns float64 which is the mean value, or zero if the count is zero.
func histogramMeanFloat64(dp metricdata.HistogramDataPoint[float64]) float64 {
	if dp.Count > 0 {
		return dp.Sum / float64(dp.Count)
	}
	return 0
}

// attributeSetToMap converts an attribute.Set to a map of strings.
//
// Takes attrs (attribute.Set) which contains the attributes to convert.
//
// Returns map[string]string which maps attribute keys to their string values.
func attributeSetToMap(attrs attribute.Set) map[string]string {
	result := make(map[string]string)
	iterator := attrs.Iter()
	for iterator.Next() {
		kv := iterator.Attribute()
		result[string(kv.Key)] = kv.Value.Emit()
	}
	return result
}
