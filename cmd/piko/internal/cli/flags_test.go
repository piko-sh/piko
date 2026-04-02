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

package cli

import (
	"testing"
	"time"
)

func TestParseGlobalFlags_Defaults(t *testing.T) {
	t.Parallel()

	opts, remaining := parseGlobalFlags([]string{})

	if opts.Endpoint != defaultEndpoint {
		t.Errorf("Endpoint = %q, want %q", opts.Endpoint, defaultEndpoint)
	}
	if opts.Output != defaultOutputFormat {
		t.Errorf("Output = %q, want %q", opts.Output, defaultOutputFormat)
	}
	if opts.Timeout != defaultTimeout {
		t.Errorf("Timeout = %v, want %v", opts.Timeout, defaultTimeout)
	}
	if opts.NoColour {
		t.Error("NoColour should be false by default")
	}
	if opts.NoHeaders {
		t.Error("NoHeaders should be false by default")
	}
	if len(remaining) != 0 {
		t.Errorf("remaining = %v, want empty", remaining)
	}
}

func TestParseGlobalFlags_LongFlags(t *testing.T) {
	t.Parallel()

	opts, remaining := parseGlobalFlags([]string{
		"--endpoint", "localhost:1234",
		"--output", "json",
		"--timeout", "10s",
		"--no-colour",
		"health",
	})

	if opts.Endpoint != "localhost:1234" {
		t.Errorf("Endpoint = %q, want %q", opts.Endpoint, "localhost:1234")
	}
	if opts.Output != "json" {
		t.Errorf("Output = %q, want %q", opts.Output, "json")
	}
	if opts.Timeout != 10*time.Second {
		t.Errorf("Timeout = %v, want %v", opts.Timeout, 10*time.Second)
	}
	if !opts.NoColour {
		t.Error("NoColour should be true")
	}
	if len(remaining) != 1 || remaining[0] != "health" {
		t.Errorf("remaining = %v, want [health]", remaining)
	}
}

func TestParseGlobalFlags_ShortFlags(t *testing.T) {
	t.Parallel()

	opts, remaining := parseGlobalFlags([]string{
		"-e", "10.0.0.1:9091",
		"-o", "json",
		"-t", "3s",
		"tasks",
	})

	if opts.Endpoint != "10.0.0.1:9091" {
		t.Errorf("Endpoint = %q, want %q", opts.Endpoint, "10.0.0.1:9091")
	}
	if opts.Output != "json" {
		t.Errorf("Output = %q, want %q", opts.Output, "json")
	}
	if opts.Timeout != 3*time.Second {
		t.Errorf("Timeout = %v, want %v", opts.Timeout, 3*time.Second)
	}
	if len(remaining) != 1 || remaining[0] != "tasks" {
		t.Errorf("remaining = %v, want [tasks]", remaining)
	}
}

func TestParseGlobalFlags_RemainingArgs(t *testing.T) {
	t.Parallel()

	_, remaining := parseGlobalFlags([]string{"health", "--verbose"})

	if len(remaining) != 2 {
		t.Errorf("remaining has %d elements, want 2", len(remaining))
	}
}

func TestParseGlobalFlags_Raw(t *testing.T) {
	t.Parallel()

	opts, _ := parseGlobalFlags([]string{"--raw", "health"})

	if !opts.NoColour {
		t.Error("NoColour should be true when --raw is set")
	}
}

func TestParseGlobalFlags_Wide(t *testing.T) {
	t.Parallel()

	opts, _ := parseGlobalFlags([]string{"-o", "wide", "health"})

	if opts.Output != "wide" {
		t.Errorf("Output = %q, want %q", opts.Output, "wide")
	}
}

func TestParseGlobalFlags_NoHeaders(t *testing.T) {
	t.Parallel()

	opts, _ := parseGlobalFlags([]string{"--no-headers", "health"})

	if !opts.NoHeaders {
		t.Error("NoHeaders should be true when --no-headers is set")
	}
}

func TestParseGlobalFlags_InterspersedOutput(t *testing.T) {
	t.Parallel()

	opts, remaining := parseGlobalFlags([]string{"health", "-o", "wide"})

	if opts.Output != "wide" {
		t.Errorf("Output = %q, want %q", opts.Output, "wide")
	}
	if len(remaining) != 1 || remaining[0] != "health" {
		t.Errorf("remaining = %v, want [health]", remaining)
	}
}

func TestParseGlobalFlags_InterspersedJSON(t *testing.T) {
	t.Parallel()

	opts, remaining := parseGlobalFlags([]string{"health", "Liveness", "-o", "json"})

	if opts.Output != "json" {
		t.Errorf("Output = %q, want %q", opts.Output, "json")
	}
	if len(remaining) != 2 || remaining[0] != "health" || remaining[1] != "Liveness" {
		t.Errorf("remaining = %v, want [health Liveness]", remaining)
	}
}

func TestParseGlobalFlags_InterspersedBoolFlags(t *testing.T) {
	t.Parallel()

	opts, remaining := parseGlobalFlags([]string{"health", "--raw", "--no-headers"})

	if !opts.NoColour {
		t.Error("NoColour should be true")
	}
	if !opts.NoHeaders {
		t.Error("NoHeaders should be true")
	}
	if len(remaining) != 1 || remaining[0] != "health" {
		t.Errorf("remaining = %v, want [health]", remaining)
	}
}

func TestParseGlobalFlags_EqualsSyntax(t *testing.T) {
	t.Parallel()

	opts, remaining := parseGlobalFlags([]string{"health", "-o=json"})

	if opts.Output != "json" {
		t.Errorf("Output = %q, want %q", opts.Output, "json")
	}
	if len(remaining) != 1 || remaining[0] != "health" {
		t.Errorf("remaining = %v, want [health]", remaining)
	}
}

func TestParseGlobalFlags_UnknownFlagsPassThrough(t *testing.T) {
	t.Parallel()

	_, remaining := parseGlobalFlags([]string{"health", "--verbose", "--errors"})

	if len(remaining) != 3 {
		t.Errorf("remaining has %d elements, want 3 (all unknown flags pass through)", len(remaining))
	}
}

func TestSeparateGlobalFlags(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name            string
		arguments       []string
		wantGlobalCount int
		wantRemainCount int
	}{
		{name: "empty", arguments: nil, wantGlobalCount: 0, wantRemainCount: 0},
		{name: "no global flags", arguments: []string{"health", "Liveness"}, wantGlobalCount: 0, wantRemainCount: 2},
		{name: "output after positional", arguments: []string{"health", "-o", "wide"}, wantGlobalCount: 2, wantRemainCount: 1},
		{name: "bool flag after positional", arguments: []string{"health", "--raw"}, wantGlobalCount: 1, wantRemainCount: 1},
		{name: "mixed with limit", arguments: []string{"health", "-o", "wide", "--limit", "10"}, wantGlobalCount: 4, wantRemainCount: 1},
		{name: "mixed with resource flags", arguments: []string{"health", "-o", "wide", "--errors"}, wantGlobalCount: 2, wantRemainCount: 2},
		{name: "equals syntax", arguments: []string{"health", "--output=json"}, wantGlobalCount: 1, wantRemainCount: 1},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			global, remaining := separateGlobalFlags(tc.arguments)
			if len(global) != tc.wantGlobalCount {
				t.Errorf("global flags = %v (len %d), want len %d", global, len(global), tc.wantGlobalCount)
			}
			if len(remaining) != tc.wantRemainCount {
				t.Errorf("remaining = %v (len %d), want len %d", remaining, len(remaining), tc.wantRemainCount)
			}
		})
	}
}

func TestParseGlobalFlags_Limit(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		arguments []string
		expected  int
	}{
		{name: "default when no flag", arguments: []string{"health"}, expected: 0},
		{name: "long flag", arguments: []string{"--limit", "5", "health"}, expected: 5},
		{name: "short flag", arguments: []string{"-n", "10", "health"}, expected: 10},
		{name: "interspersed", arguments: []string{"tasks", "--limit", "15"}, expected: 15},
		{name: "short interspersed", arguments: []string{"tasks", "-n", "3"}, expected: 3},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			opts, _ := parseGlobalFlags(tc.arguments)
			if opts.Limit != tc.expected {
				t.Errorf("Limit = %d, want %d", opts.Limit, tc.expected)
			}
		})
	}
}

func TestParseGlobalFlags_Certs(t *testing.T) {
	t.Parallel()

	opts, remaining := parseGlobalFlags([]string{"--certs", "/path/to/certs", "health"})

	if opts.CertsDir != "/path/to/certs" {
		t.Errorf("CertsDir = %q, want %q", opts.CertsDir, "/path/to/certs")
	}
	if len(remaining) != 1 || remaining[0] != "health" {
		t.Errorf("remaining = %v, want [health]", remaining)
	}
}

func TestParseGlobalFlags_CertsInterspersed(t *testing.T) {
	t.Parallel()

	opts, remaining := parseGlobalFlags([]string{"diagnostics", "--certs", "/certs"})

	if opts.CertsDir != "/certs" {
		t.Errorf("CertsDir = %q, want %q", opts.CertsDir, "/certs")
	}
	if len(remaining) != 1 || remaining[0] != "diagnostics" {
		t.Errorf("remaining = %v, want [diagnostics]", remaining)
	}
}

func TestParseGlobalFlags_CertsEquals(t *testing.T) {
	t.Parallel()

	opts, remaining := parseGlobalFlags([]string{"health", "--certs=/my/certs"})

	if opts.CertsDir != "/my/certs" {
		t.Errorf("CertsDir = %q, want %q", opts.CertsDir, "/my/certs")
	}
	if len(remaining) != 1 || remaining[0] != "health" {
		t.Errorf("remaining = %v, want [health]", remaining)
	}
}

func TestParseGlobalFlags_CertsDefault_Empty(t *testing.T) {
	t.Parallel()

	opts, _ := parseGlobalFlags([]string{"health"})

	if opts.CertsDir != "" {
		t.Errorf("CertsDir = %q, want empty", opts.CertsDir)
	}
}
