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

// Package metrics_exporter_prometheus implements the metrics
// exporter interface using Prometheus.
//
// This package bridges OpenTelemetry metrics to the Prometheus
// exposition format. It creates a dedicated Prometheus registry
// and exposes an HTTP handler suitable for scraping. Use [New]
// for explicit error handling, or [MustNew] when a failure to
// create the exporter should panic. All methods are safe for
// concurrent use.
package metrics_exporter_prometheus
