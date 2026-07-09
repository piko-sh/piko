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

// Package readiness_collector_grpcfb periodically samples piko's readiness tree and
// forwards it as telemetry_grpcfb MetricPoints onto a shared telemetry client.
//
// It samples the same dependency health piko shows its TUI/CLI/devwidget via
// CheckReadiness, so an external monitor's Dependencies screen reads live health over the
// SAME stream as analytics + spans + logs + runtime gauges, without a second connection.
//
// Each readiness dependency becomes ONE MetricPoint per sample, discriminated by
// Kind="dependency" so the sink filters these out of normal gauges:
//
//	Name        = dependency name
//	Kind        = "dependency"
//	Value       = check latency in ms (0 when the duration is unparseable)
//	Unit        = "ms"
//	TimestampMs = sample time
//	Labels      = {status: healthy|degraded|down, message: ..., icon: ...}
//
// The collector reads readiness through the public readiness.Probe seam (SSRServer.
// HealthProbe), so it never imports piko's internal monitoring packages.
package readiness_collector_grpcfb
