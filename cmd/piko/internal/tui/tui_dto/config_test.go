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

package tui_dto

import "testing"

func TestConfig_GetClock_NilDefault(t *testing.T) {
	c := &Config{}
	clk := c.GetClock()
	if clk == nil {
		t.Error("GetClock() should return non-nil clock when Clock is nil")
	}
}

func TestConfig_GetClock_Configured(t *testing.T) {
	definition := DefaultConfig()
	clk := definition.GetClock()
	if clk == nil {
		t.Error("GetClock() should return the configured clock")
	}
}

func TestConfig_HasEndpoints(t *testing.T) {
	c := &Config{}
	if c.HasPikoEndpoint() {
		t.Error("empty config should not have Piko endpoint")
	}
	if c.HasHealthEndpoint() {
		t.Error("empty config should not have health endpoint")
	}
	if c.HasMonitoringEndpoint() {
		t.Error("empty config should not have monitoring endpoint")
	}
	if c.HasPrometheus() {
		t.Error("empty config should not have Prometheus")
	}
	if c.HasJaeger() {
		t.Error("empty config should not have Jaeger")
	}

	definition := DefaultConfig()
	if !definition.HasPikoEndpoint() {
		t.Error("default config should have Piko endpoint")
	}
	if !definition.HasHealthEndpoint() {
		t.Error("default config should have health endpoint")
	}
	if !definition.HasMonitoringEndpoint() {
		t.Error("default config should have monitoring endpoint")
	}
}

func TestConfig_HasAnyOTELSource(t *testing.T) {
	c := &Config{}
	if c.HasAnyOTELSource() {
		t.Error("empty config should not have any OTEL source")
	}

	c.PikoEndpoint = "http://localhost:8080"
	if !c.HasAnyOTELSource() {
		t.Error("config with Piko endpoint should have OTEL source")
	}

	c2 := &Config{PrometheusURL: "http://prom:9090"}
	if !c2.HasAnyOTELSource() {
		t.Error("config with Prometheus should have OTEL source")
	}

	c3 := &Config{JaegerURL: "http://jaeger:16686"}
	if !c3.HasAnyOTELSource() {
		t.Error("config with Jaeger should have OTEL source")
	}
}
