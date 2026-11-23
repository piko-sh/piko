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
package metrics_exporter_prometheus_test

import (
	sdk "go.opentelemetry.io/otel/sdk/metric"
	"net/http/httptest"
	exporter "piko.sh/piko/wdk/metrics/metrics_exporter_prometheus"
	"regexp"
	"testing"
)

func TestPublicMetricsContract(t *testing.T) {
	e, err := exporter.New()
	if err != nil {
		t.Fatal(err)
	}
	reader, ok := e.Reader().(sdk.Reader)
	if !ok {
		t.Fatal("PIKO_CHECK reader: incompatible OTEL metric reader")
	}
	p := sdk.NewMeterProvider(sdk.WithReader(reader))
	defer p.Shutdown(t.Context())
	g, err := p.Meter("local").Int64Gauge("fuzz_balance")
	if err != nil {
		t.Fatal(err)
	}
	g.Record(t.Context(), 7)
	r := httptest.NewRecorder()
	e.Handler().ServeHTTP(r, httptest.NewRequest("GET", "/metrics", nil))
	if r.Code != 200 || !regexp.MustCompile(`(?m)^fuzz_balance(?:\{[^\n]*\})? 7$`).MatchString(r.Body.String()) {
		t.Fatalf("PIKO_CHECK scrape: status %d, %q", r.Code, r.Body.String())
	}
}
