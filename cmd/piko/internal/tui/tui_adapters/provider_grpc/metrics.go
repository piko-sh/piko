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

package provider_grpc

import (
	"context"
	"fmt"
	"sync"
	"time"

	"piko.sh/piko/cmd/piko/internal/tui/tui_domain"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/logger"
	pb "piko.sh/piko/wdk/monitoring/monitoring_api/gen"
)

var _ tui_domain.MetricsProvider = (*MetricsProvider)(nil)

// MetricsProvider provides metrics data through a gRPC connection.
// It implements tui_domain.MetricsProvider and is safe for concurrent use.
type MetricsProvider struct {
	// conn holds the gRPC connection with health and metrics clients.
	conn *Connection

	// metrics maps metric names to their time series data.
	metrics map[string]*tui_domain.MetricSeries

	// metricNames holds the sorted list of metric names for ListMetrics.
	metricNames []string

	// mu guards access to metrics and metricNames during concurrent operations.
	mu sync.RWMutex

	// interval is the refresh interval between metrics updates.
	interval time.Duration
}

// NewMetricsProvider creates a new MetricsProvider.
//
// Takes conn (*Connection) which is the shared gRPC connection.
// Takes interval (time.Duration) which is the refresh interval.
//
// Returns *MetricsProvider which is the configured provider.
func NewMetricsProvider(conn *Connection, interval time.Duration) *MetricsProvider {
	return &MetricsProvider{
		conn:        conn,
		metrics:     make(map[string]*tui_domain.MetricSeries),
		metricNames: nil,
		mu:          sync.RWMutex{},
		interval:    interval,
	}
}

// Name returns the provider name.
//
// Returns string which is the identifier for this metrics provider.
func (*MetricsProvider) Name() string {
	return "grpc-metrics"
}

// Health checks if the gRPC connection is healthy.
//
// Returns error when the health check request fails.
func (p *MetricsProvider) Health(ctx context.Context) error {
	_, err := p.conn.healthClient.GetHealth(ctx, &pb.GetHealthRequest{})
	if err != nil {
		return fmt.Errorf("checking metrics provider health via gRPC: %w", err)
	}
	return nil
}

// Close releases resources.
//
// Returns error when resource cleanup fails.
func (*MetricsProvider) Close() error {
	return nil
}

// RefreshInterval returns the refresh interval.
//
// Returns time.Duration which is the interval between metric refreshes.
func (p *MetricsProvider) RefreshInterval() time.Duration {
	return p.interval
}

// Refresh fetches the latest metrics via gRPC.
//
// Returns error when the gRPC call fails.
//
// Safe for concurrent use; guards internal state with a mutex.
func (p *MetricsProvider) Refresh(ctx context.Context) error {
	ctx, l := logger_domain.From(ctx, log)

	return instrumentedCall(ctx, func() error {
		response, err := p.conn.metricsClient.GetMetrics(ctx, &pb.GetMetricsRequest{})
		if err != nil {
			l.Debug("Failed to fetch metrics", logger.Error(err))
			return fmt.Errorf("fetching metrics: %w", err)
		}

		metrics, names := convertMetrics(response)

		p.mu.Lock()
		p.metrics = metrics
		p.metricNames = names
		p.mu.Unlock()

		return nil
	})
}

// ListMetrics returns available metric names.
//
// Returns []string which contains a copy of all registered metric names.
// Returns error which is always nil.
//
// Safe for concurrent use; guards access with a read lock.
func (p *MetricsProvider) ListMetrics(_ context.Context) ([]string, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	result := make([]string, len(p.metricNames))
	copy(result, p.metricNames)
	return result, nil
}

// Query fetches a metric series for the given time range.
//
// Takes metric (string) which is the name of the metric to look up.
//
// Returns *tui_domain.MetricSeries which contains the time series
// data for the requested metric.
// Returns error when the metric is not found.
//
// Safe for concurrent use; protected by a read lock.
func (p *MetricsProvider) Query(_ context.Context, metric string, _, _ time.Time) (*tui_domain.MetricSeries, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	series, ok := p.metrics[metric]
	if !ok {
		return nil, fmt.Errorf("metric %q not found", metric)
	}

	return series, nil
}

// Current fetches the current value of a metric.
//
// Takes metric (string) which specifies the name of the metric to retrieve.
//
// Returns *tui_domain.MetricValue which contains the most recent value.
// Returns error when the metric is not found.
//
// Safe for concurrent use; protected by a read lock.
func (p *MetricsProvider) Current(_ context.Context, metric string) (*tui_domain.MetricValue, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	series, ok := p.metrics[metric]
	if !ok {
		return nil, fmt.Errorf("metric %q not found", metric)
	}

	return series.Latest(), nil
}

// convertMetrics converts protobuf metrics to domain format.
//
// Takes response (*pb.GetMetricsResponse) which contains the
// protobuf metrics data.
//
// Returns map[string]*tui_domain.MetricSeries which maps metric names to their
// series data.
// Returns []string which contains the ordered list of metric names.
func convertMetrics(response *pb.GetMetricsResponse) (map[string]*tui_domain.MetricSeries, []string) {
	metrics := make(map[string]*tui_domain.MetricSeries)
	names := make([]string, 0, len(response.GetMetrics()))

	for _, m := range response.GetMetrics() {
		name := m.GetName()
		names = append(names, name)

		values := make([]tui_domain.MetricValue, 0, len(m.GetDataPoints()))
		for _, dp := range m.GetDataPoints() {
			values = append(values, tui_domain.MetricValue{
				Timestamp: time.UnixMilli(dp.GetTimestampMs()),
				Value:     dp.GetValue(),
				Labels:    dp.GetAttributes(),
			})
		}

		series := &tui_domain.MetricSeries{
			Name:        name,
			Unit:        m.GetUnit(),
			Description: m.GetDescription(),
			Values:      values,
		}

		metrics[name] = series
	}

	return metrics, names
}
