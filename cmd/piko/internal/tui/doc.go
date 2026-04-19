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

// Package tui provides a terminal user interface for monitoring Piko
// applications.
//
// This package is the public facade for Piko's TUI subsystem. It
// displays real-time information about registry artefacts,
// orchestrator tasks, metrics, traces, health status, and system
// statistics. The TUI connects to a running Piko instance via its
// gRPC monitoring endpoint.
//
// # Quick start
//
// The TUI connects to your Piko application's gRPC monitoring
// endpoint:
//
//	t, err := tui.New(
//	    tui.WithMonitoringEndpoint("localhost:9091"),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer t.Close()
//
//	if err := t.Run(ctx); err != nil {
//	    log.Fatal(err)
//	}
//
// # Configuration
//
// The TUI is configured using functional options:
//
//   - [WithMonitoringEndpoint]: connect to Piko's gRPC
//     monitoring server
//   - [WithPikoEndpoint]: connect to Piko's HTTP endpoint
//   - [WithPrometheus]: optionally connect to external Prometheus
//   - [WithJaeger]: optionally connect to external Jaeger
//   - [WithRefreshInterval]: set data refresh interval
//     (default 2s)
//   - [WithTheme]: select UI theme ("default" or "minimal")
//   - [WithTitle]: set the TUI title bar text
//   - [WithConfig]: apply settings from a Config value
//
// Configuration can also be loaded from a tui.yaml file using
// [LoadConfig], which checks ./tui.yaml and
// $HOME/.config/piko/tui.yaml. PIKO_TUI_* environment variables
// override file values. The TUI's loader uses the same generic
// config_domain machinery exposed at piko.sh/piko/wdk/config.
//
// # Custom panels and providers
//
// The TUI can be extended with custom panels and data providers:
//
//	t, _ := tui.New(
//	    tui.WithMonitoringEndpoint("localhost:9091"),
//	    tui.WithPanel(&OrdersPanel{}),
//	    tui.WithMetricsProvider(&CustomMetrics{}),
//	)
//
// # Keybindings
//
// Navigation:
//   - Tab / Shift+Tab: switch between panels
//   - 1-9: jump to panel by number
//   - j/k or arrows: navigate within panels
//   - Enter: expand/select item
//   - Esc: collapse/back
//   - ?: toggle help
//   - q: quit
//
// # Diagnostics
//
// Use [RunDiagnostics] to test connectivity to the gRPC
// monitoring endpoint before launching the full TUI.
//
// # Thread safety
//
// A [TUI] instance must not be shared between goroutines. Call
// [TUI.Run] from a single goroutine; it blocks until the user
// exits or the context is cancelled.
package tui
