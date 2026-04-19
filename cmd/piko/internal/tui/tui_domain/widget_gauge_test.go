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

package tui_domain

import (
	"strings"
	"testing"
)

func TestGaugeRendersToTargetWidth(t *testing.T) {
	config := GaugeConfig{
		Label:    "heap",
		Width:    40,
		Used:     50,
		Max:      100,
		ShowText: true,
	}
	out := Gauge(config)
	if w := TextWidth(out); w != 40 {
		t.Errorf("Gauge width = %d, want 40 (output %q)", w, out)
	}
}

func TestGaugeShowsUsedAndMax(t *testing.T) {
	out := Gauge(GaugeConfig{
		Width:    50,
		Used:     25,
		Max:      100,
		ShowText: true,
	})
	if !strings.Contains(out, "25") || !strings.Contains(out, "100") {
		t.Errorf("expected used/max text in gauge: %q", out)
	}
}

func TestGaugeEmptyAtZero(t *testing.T) {
	out := Gauge(GaugeConfig{
		Width: 30,
		Used:  0,
		Max:   100,
	})
	if strings.Contains(out, string(gaugeDefaultFill)) {
		t.Errorf("zero-used gauge should not contain fill char: %q", out)
	}
}

func TestGaugeFullAtMax(t *testing.T) {
	out := Gauge(GaugeConfig{
		Width: 30,
		Used:  100,
		Max:   100,
	})
	if strings.Contains(out, string(gaugeDefaultEmpty)) {
		t.Errorf("full gauge should not contain empty char: %q", out)
	}
}

func TestGaugeMaxZeroRendersDash(t *testing.T) {
	out := Gauge(GaugeConfig{
		Width:    20,
		Used:     5,
		Max:      0,
		ShowText: true,
	})
	if !strings.Contains(out, "-") {
		t.Errorf("zero-max gauge should contain dash: %q", out)
	}
}

func TestSeverityFromPercent(t *testing.T) {
	cases := []struct {
		in   float64
		want Severity
	}{
		{0.1, SeverityHealthy},
		{0.65, SeverityWarning},
		{0.85, SeverityCritical},
		{1.0, SeveritySaturated},
		{1.5, SeveritySaturated},
	}
	for _, c := range cases {
		if got := severityFromPercent(c.in); got != c.want {
			t.Errorf("severityFromPercent(%.2f) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestGaugeZeroWidthSafe(t *testing.T) {
	if out := Gauge(GaugeConfig{Width: 0}); out != "" {
		t.Errorf("zero-width gauge = %q, want empty", out)
	}
}
