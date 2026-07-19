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

	"github.com/stretchr/testify/assert"
)

func TestParseGlobalFlags_Defaults(t *testing.T) {
	t.Parallel()

	opts, remaining := parseGlobalFlags([]string{})

	assert.Equal(t, defaultEndpoint, opts.Endpoint, "Endpoint = %q, want %q", opts.Endpoint, defaultEndpoint)
	assert.Equal(t, defaultOutputFormat, opts.Output, "Output = %q, want %q", opts.Output, defaultOutputFormat)
	assert.Equal(t, defaultTimeout, opts.Timeout, "Timeout = %v, want %v", opts.Timeout, defaultTimeout)
	assert.False(t, opts.NoColour, "NoColour should be false by default")
	assert.False(t, opts.NoHeaders, "NoHeaders should be false by default")
	assert.Empty(t, remaining, "remaining = %v, want empty", remaining)
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

	assert.Equal(t, "localhost:1234", opts.Endpoint, "Endpoint = %q, want %q", opts.Endpoint, "localhost:1234")
	assert.Equal(t, "json", opts.Output, "Output = %q, want %q", opts.Output, "json")
	assert.Equal(t, 10*time.Second, opts.Timeout, "Timeout = %v, want %v", opts.Timeout, 10*time.Second)
	assert.True(t, opts.NoColour, "NoColour should be true")
	assert.False(t, len(remaining) != 1 || remaining[0] != "health", "remaining = %v, want [health]", remaining)
}

func TestParseGlobalFlags_ShortFlags(t *testing.T) {
	t.Parallel()

	opts, remaining := parseGlobalFlags([]string{
		"-e", "10.0.0.1:9091",
		"-o", "json",
		"-t", "3s",
		"tasks",
	})

	assert.Equal(t, "10.0.0.1:9091", opts.Endpoint, "Endpoint = %q, want %q", opts.Endpoint, "10.0.0.1:9091")
	assert.Equal(t, "json", opts.Output, "Output = %q, want %q", opts.Output, "json")
	assert.Equal(t, 3*time.Second, opts.Timeout, "Timeout = %v, want %v", opts.Timeout, 3*time.Second)
	assert.False(t, len(remaining) != 1 || remaining[0] != "tasks", "remaining = %v, want [tasks]", remaining)
}

func TestParseGlobalFlags_RemainingArgs(t *testing.T) {
	t.Parallel()

	_, remaining := parseGlobalFlags([]string{"health", "--verbose"})

	assert.Len(t, remaining, 2, "remaining has %d elements, want 2", len(remaining))
}

func TestParseGlobalFlags_Raw(t *testing.T) {
	t.Parallel()

	opts, _ := parseGlobalFlags([]string{"--raw", "health"})

	assert.True(t, opts.NoColour, "NoColour should be true when --raw is set")
}

func TestParseGlobalFlags_Wide(t *testing.T) {
	t.Parallel()

	opts, _ := parseGlobalFlags([]string{"-o", "wide", "health"})

	assert.Equal(t, "wide", opts.Output, "Output = %q, want %q", opts.Output, "wide")
}

func TestParseGlobalFlags_NoHeaders(t *testing.T) {
	t.Parallel()

	opts, _ := parseGlobalFlags([]string{"--no-headers", "health"})

	assert.True(t, opts.NoHeaders, "NoHeaders should be true when --no-headers is set")
}

func TestParseGlobalFlags_InterspersedOutput(t *testing.T) {
	t.Parallel()

	opts, remaining := parseGlobalFlags([]string{"health", "-o", "wide"})

	assert.Equal(t, "wide", opts.Output, "Output = %q, want %q", opts.Output, "wide")
	assert.False(t, len(remaining) != 1 || remaining[0] != "health", "remaining = %v, want [health]", remaining)
}

func TestParseGlobalFlags_InterspersedJSON(t *testing.T) {
	t.Parallel()

	opts, remaining := parseGlobalFlags([]string{"health", "Liveness", "-o", "json"})

	assert.Equal(t, "json", opts.Output, "Output = %q, want %q", opts.Output, "json")
	assert.False(t, len(remaining) != 2 || remaining[0] != "health" || remaining[1] != "Liveness", "remaining = %v, want [health Liveness]", remaining)
}

func TestParseGlobalFlags_InterspersedBoolFlags(t *testing.T) {
	t.Parallel()

	opts, remaining := parseGlobalFlags([]string{"health", "--raw", "--no-headers"})

	assert.True(t, opts.NoColour, "NoColour should be true")
	assert.True(t, opts.NoHeaders, "NoHeaders should be true")
	assert.False(t, len(remaining) != 1 || remaining[0] != "health", "remaining = %v, want [health]", remaining)
}

func TestParseGlobalFlags_EqualsSyntax(t *testing.T) {
	t.Parallel()

	opts, remaining := parseGlobalFlags([]string{"health", "-o=json"})

	assert.Equal(t, "json", opts.Output, "Output = %q, want %q", opts.Output, "json")
	assert.False(t, len(remaining) != 1 || remaining[0] != "health", "remaining = %v, want [health]", remaining)
}

func TestParseGlobalFlags_UnknownFlagsPassThrough(t *testing.T) {
	t.Parallel()

	_, remaining := parseGlobalFlags([]string{"health", "--verbose", "--errors"})

	assert.Len(t, remaining, 3, "remaining has %d elements, want 3 (all unknown flags pass through)", len(remaining))
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
			assert.Len(t, global, tc.wantGlobalCount, "global flags = %v (len %d), want len %d", global, len(global), tc.wantGlobalCount)
			assert.Len(t, remaining, tc.wantRemainCount, "remaining = %v (len %d), want len %d", remaining, len(remaining), tc.wantRemainCount)
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
			assert.Equal(t, tc.expected, opts.Limit, "Limit = %d, want %d", opts.Limit, tc.expected)
		})
	}
}

func TestParseGlobalFlags_Certs(t *testing.T) {
	t.Parallel()

	opts, remaining := parseGlobalFlags([]string{"--certs", "/path/to/certs", "health"})

	assert.Equal(t, "/path/to/certs", opts.CertsDir, "CertsDir = %q, want %q", opts.CertsDir, "/path/to/certs")
	assert.False(t, len(remaining) != 1 || remaining[0] != "health", "remaining = %v, want [health]", remaining)
}

func TestParseGlobalFlags_CertsInterspersed(t *testing.T) {
	t.Parallel()

	opts, remaining := parseGlobalFlags([]string{"diagnostics", "--certs", "/certs"})

	assert.Equal(t, "/certs", opts.CertsDir, "CertsDir = %q, want %q", opts.CertsDir, "/certs")
	assert.False(t, len(remaining) != 1 || remaining[0] != "diagnostics", "remaining = %v, want [diagnostics]", remaining)
}

func TestParseGlobalFlags_CertsEquals(t *testing.T) {
	t.Parallel()

	opts, remaining := parseGlobalFlags([]string{"health", "--certs=/my/certs"})

	assert.Equal(t, "/my/certs", opts.CertsDir, "CertsDir = %q, want %q", opts.CertsDir, "/my/certs")
	assert.False(t, len(remaining) != 1 || remaining[0] != "health", "remaining = %v, want [health]", remaining)
}

func TestParseGlobalFlags_CertsDefault_Empty(t *testing.T) {
	t.Parallel()

	opts, _ := parseGlobalFlags([]string{"health"})

	assert.Empty(t, opts.CertsDir, "CertsDir = %q, want empty", opts.CertsDir)
}
