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

package metrics_exporter_prometheus

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	prometheusexporter "go.opentelemetry.io/otel/exporters/prometheus"
	"piko.sh/piko/internal/monitoring/monitoring_domain"
	"piko.sh/piko/wdk/metrics"
)

var _ metrics.Exporter = (*Exporter)(nil)

// Exporter wraps the OpenTelemetry Prometheus exporter.
// It provides a metric reader for the MeterProvider and an HTTP handler for
// the /metrics endpoint.
type Exporter struct {
	// exporter is the Prometheus metrics exporter.
	exporter *prometheusexporter.Exporter

	// registry holds the Prometheus registry for collecting metrics.
	registry *prometheus.Registry
}

// Reader returns the OTEL metric reader that should be registered
// with the MeterProvider. Metrics recorded through OTEL will be
// exposed via the Handler().
//
// Returns monitoring_domain.MetricReader for MeterProvider registration.
func (e *Exporter) Reader() monitoring_domain.MetricReader {
	return e.exporter
}

// Handler returns an HTTP handler that serves Prometheus metrics.
// This handler should be mounted at the configured metrics path
// (typically /metrics).
//
// Returns http.Handler which serves Prometheus-formatted metrics.
func (e *Exporter) Handler() http.Handler {
	return promhttp.HandlerFor(e.registry, promhttp.HandlerOpts{
		EnableOpenMetrics: true,
	})
}

// New creates a Prometheus metrics exporter.
//
// Returns metrics.Exporter which is ready for use with piko.WithMetricsExporter.
// Returns error when the exporter cannot be created.
func New() (metrics.Exporter, error) {
	registry := prometheus.NewRegistry()

	exporter, err := prometheusexporter.New(
		prometheusexporter.WithRegisterer(registry),
		prometheusexporter.WithoutTargetInfo(),
	)
	if err != nil {
		return nil, err
	}

	return &Exporter{
		exporter: exporter,
		registry: registry,
	}, nil
}

// MustNew creates a new Prometheus metrics exporter.
//
// Returns metrics.Exporter which is ready for use with piko.WithMetricsExporter.
//
// Panics when the exporter cannot be created.
func MustNew() metrics.Exporter {
	exp, err := New()
	if err != nil {
		panic("metrics_exporter_prometheus: failed to create exporter: " + err.Error())
	}
	return exp
}
